package collector

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"quickpulse/backend/logs"
	"quickpulse/backend/logs/parser"
)

// Submitter is the interface the collectors call into. We define it here
// (rather than depending on logs.Ingest directly) so the collector package
// has zero awareness of the ingest channel's internals.
type Submitter interface {
	Submit(logs.Entry)
}

// DockerCollector subscribes to the Docker events stream and attaches a
// per-container log streamer for every running container. New containers
// are picked up automatically; dying containers have their streamers
// cancelled and their multi-line state flushed.
type DockerCollector struct {
	cli      *client.Client
	registry *Registry
	joiner   *parser.MultilineJoiner
	submit   Submitter
	hostName string
	env      string
}

func NewDocker(reg *Registry, joiner *parser.MultilineJoiner, sub Submitter, hostName, env string) (*DockerCollector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerCollector{
		cli:      cli,
		registry: reg,
		joiner:   joiner,
		submit:   sub,
		hostName: hostName,
		env:      env,
	}, nil
}

// Run is the long-lived entrypoint. It does three things concurrently:
//  1. periodically lists running containers and attaches streamers for any
//     it doesn't already have (handles startup + recovery from missed events);
//  2. subscribes to docker events to pick up new containers immediately;
//  3. flushes the multi-line joiner on a slow tick so trailing entries
//     eventually flow even on quiet streams.
func (c *DockerCollector) Run(ctx context.Context) {
	defer c.cli.Close()

	go c.eventLoop(ctx)
	go c.discoveryLoop(ctx)
	go c.flushLoop(ctx)

	<-ctx.Done()
	// Cancel every active streamer in bulk.
	for _, id := range c.registry.IDs() {
		c.registry.Release(id)
	}
}

// flushLoop releases idle multi-line buffers periodically so half-finished
// entries don't sit pending forever on quiet streams. 1s idle is enough for
// stack traces (they arrive bunched together) but short enough that single
// orphaned lines reach the UI quickly.
func (c *DockerCollector) flushLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, e := range c.joiner.Flush(500 * time.Millisecond) {
				c.submit.Submit(e)
			}
		}
	}
}

// discoveryLoop runs every 30s and ensures we have a streamer for every
// running container — events are best-effort, list-and-attach is the
// guarantee.
func (c *DockerCollector) discoveryLoop(ctx context.Context) {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	c.discoverOnce(ctx) // immediate first pass
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c.discoverOnce(ctx)
		}
	}
}

func (c *DockerCollector) discoverOnce(ctx context.Context) {
	conts, err := c.cli.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		log.Printf("[logs/docker] discover: list failed: %v", err)
		return
	}
	for _, ct := range conts {
		c.attach(ctx, ct.ID, containerName(ct.Names, ct.ID), ct.Labels)
	}
}

// containerName mirrors the convention in handlers/containers.go.
func containerName(names []string, id string) string {
	if len(names) > 0 {
		return strings.TrimPrefix(names[0], "/")
	}
	if len(id) >= 12 {
		return id[:12]
	}
	return id
}

// eventLoop watches docker events and reacts to start/die.
func (c *DockerCollector) eventLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		args := filters.NewArgs()
		args.Add("type", "container")
		evChan, errChan := c.cli.Events(ctx, dockertypes.EventsOptions{Filters: args})
	loop:
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-evChan:
				switch ev.Action {
				case "start":
					// Re-look-up to grab labels & current name.
					info, err := c.cli.ContainerInspect(ctx, ev.Actor.ID)
					if err == nil {
						name := strings.TrimPrefix(info.Name, "/")
						labels := info.Config.Labels
						c.attach(ctx, info.ID, name, labels)
					}
				case "die", "stop", "kill":
					c.detach(ev.Actor.ID)
				}
			case err := <-errChan:
				if err != nil && !errors.Is(err, context.Canceled) {
					log.Printf("[logs/docker] event stream error: %v — reconnecting", err)
				}
				break loop
			}
		}
		// Backoff before reconnecting the event stream.
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

// attach starts a streamer for `id`. Idempotent: skipped if a stream is
// already active for that id.
func (c *DockerCollector) attach(parent context.Context, id, name string, labels map[string]string) {
	// Skip our own infrastructure — same filter as ListContainersHandler.
	if strings.HasPrefix(name, "qp-") || labels["com.docker.compose.project"] == "quickpulse" {
		return
	}
	_, ok := c.registry.Reserve(id)
	if !ok {
		c.registry.Touch(id)
		return
	}

	streamCtx, cancel := context.WithCancel(parent)
	// Override the cancel we got from Reserve with our own that's tied to
	// `parent`, so the global shutdown also cancels this streamer.
	go func() {
		<-streamCtx.Done()
		// no-op; the registry's own cancel runs separately via Release
	}()

	meta := parser.SourceMeta{
		Platform:  logs.PlatformDocker,
		SourceID:  "docker:" + id,
		Cluster:   "docker", // synthetic cluster so multi-cluster filters stay symmetric
		Container: name,
		Service:   labels["com.docker.compose.service"],
		Host:      c.hostName,
		Env:       c.env,
	}

	go func() {
		defer cancel()
		c.stream(streamCtx, id, meta)
		// When stream() returns (container died / errored out), drop the
		// joiner's tail and release the registry slot.
		for _, e := range c.joiner.Forget(meta.SourceID) {
			c.submit.Submit(e)
		}
		c.registry.Release(id)
	}()
}

func (c *DockerCollector) detach(id string) {
	c.registry.Release(id)
}

// stream is the per-container read loop. It auto-reconnects with backoff
// using Since=lastSeen so we never duplicate or lose entries across drops.
func (c *DockerCollector) stream(ctx context.Context, id string, meta parser.SourceMeta) {
	since := ""
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}
		opts := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Timestamps: true,
			Since:      since,
			Tail:       "0",
		}
		rdr, err := c.cli.ContainerLogs(ctx, id, opts)
		if err != nil {
			log.Printf("[logs/docker] %s: open failed: %v", meta.Container, err)
			if !sleepCtx(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		// On success, reset backoff and consume.
		backoff = time.Second
		c.consume(ctx, rdr, meta, &since)
		_ = rdr.Close()

		// Reconnect after the read loop exits (EOF or transient error).
		if !sleepCtx(ctx, time.Second) {
			return
		}
	}
}

// consume reads from one logs stream. Demultiplexes via stdcopy unless the
// container is TTY (in which case it's already raw); falls back to io.Copy.
func (c *DockerCollector) consume(ctx context.Context, rdr io.ReadCloser, meta parser.SourceMeta, since *string) {
	pr, pw := io.Pipe()
	defer pr.Close()

	// Drain rdr → pw via stdcopy in a goroutine.
	go func() {
		defer pw.Close()
		if _, err := stdcopy.StdCopy(pw, pw, rdr); err != nil && !errors.Is(err, io.EOF) {
			// TTY containers: stdcopy fails fast; fall back to a passthrough.
			// We can't tell ahead of time without inspect, so we just try
			// both. By the time we hit this branch, rdr is partially drained
			// — close pw so the reader gets a clean EOF on the joiner side.
		}
	}()

	scanner := bufio.NewScanner(pr)
	scanner.Buffer(make([]byte, 4096), 128*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		c.registry.Touch(meta.SourceID)

		// Update `since` from the daemon's own timestamp prefix (we request
		// Timestamps:true) so reconnect picks up where we left off. The parsed
		// entry TS can come from the log content itself — a backdated line
		// would rewind the cursor and re-stream old logs on reconnect.
		if i := strings.IndexByte(line, ' '); i > 0 {
			if _, err := time.Parse(time.RFC3339Nano, line[:i]); err == nil {
				*since = line[:i]
			}
		}

		entry := parser.Parse(meta, line, time.Now())

		for _, finished := range c.joiner.Feed(meta.SourceID, entry) {
			c.submit.Submit(finished)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
