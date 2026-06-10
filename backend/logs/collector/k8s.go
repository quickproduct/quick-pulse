package collector

import (
	"bufio"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"quickpulse/backend/logs"
	"quickpulse/backend/logs/parser"
)

// K8sCollector enumerates every context in the user's kubeconfig and
// runs one independent watcher per cluster, so multi-cluster setups
// (e.g. one kubeconfig with `k3d-jh-local` + `k3d-quickspin-local`)
// stream into the same logs table — distinguished by a `cluster`
// dimension surfaced in the UI.
//
// We deliberately avoid the full informer (cache.SharedInformer) — it
// allocates a large in-memory store of pod objects per context, which
// we can't afford at 64 MB RSS for multiple clusters. A bare Watch
// gives us the ADDED/MODIFIED/DELETED events we need at a fraction
// of the cost.
type K8sCollector struct {
	registry *Registry
	joiner   *parser.MultilineJoiner
	submit   Submitter
	env      string
}

func NewK8s(reg *Registry, joiner *parser.MultilineJoiner, sub Submitter, env string) *K8sCollector {
	return &K8sCollector{
		registry: reg,
		joiner:   joiner,
		submit:   sub,
		env:      env,
	}
}

// Run discovers every kubeconfig context (plus in-cluster auth, if
// applicable) and launches one watch loop per cluster. New contexts
// require a restart to pick up — kubeconfigs change rarely and a
// file-watch would cost more than it saves.
func (c *K8sCollector) Run(ctx context.Context) {
	go c.flushLoop(ctx)

	// In-cluster mode (we're running inside a pod): there's only one
	// cluster to watch, and its name isn't exposed by the K8s API. We
	// label it "in-cluster".
	if cfg, err := rest.InClusterConfig(); err == nil {
		if cs, err := kubernetes.NewForConfig(cfg); err == nil {
			log.Printf("[logs/k8s] watching in-cluster context")
			go c.watchClusterLoop(ctx, "in-cluster", cs)
			return
		}
	}

	// File-based kubeconfig: enumerate every defined context and start
	// a watcher per cluster.
	path := kubeconfigPath()
	if path == "" {
		log.Printf("[logs/k8s] no kubeconfig found, k8s log collection disabled")
		return
	}
	apiCfg, err := clientcmd.LoadFromFile(path)
	if err != nil {
		log.Printf("[logs/k8s] failed to load kubeconfig %s: %v", path, err)
		return
	}
	if len(apiCfg.Contexts) == 0 {
		log.Printf("[logs/k8s] kubeconfig %s has no contexts", path)
		return
	}

	started := 0
	for ctxName := range apiCfg.Contexts {
		cs, err := clientFromContext(path, ctxName)
		if err != nil {
			log.Printf("[logs/k8s] context %q: clientset build failed: %v", ctxName, err)
			continue
		}
		log.Printf("[logs/k8s] watching context %q", ctxName)
		go c.watchClusterLoop(ctx, ctxName, cs)
		started++
	}
	if started == 0 {
		log.Printf("[logs/k8s] no contexts could be reached")
	}
}

// kubeconfigPath mirrors the lookup logic in handlers/kubernetes.go but
// without taking a dependency on that package (which would form a cycle).
func kubeconfigPath() string {
	if p := os.Getenv("KUBECONFIG"); p != "" {
		return p
	}
	if home, _ := os.UserHomeDir(); home != "" {
		p := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// clientFromContext builds a clientset for a specific kubeconfig context.
// Applies the same loopback-host rewrite trick we use in the rest of the
// codebase: when running in a container, replace 0.0.0.0/127.0.0.1/
// localhost with host.docker.internal so the API server stays reachable.
func clientFromContext(path, contextName string) (*kubernetes.Clientset, error) {
	loader := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat("/.dockerenv"); statErr == nil {
		for _, lb := range []string{"//0.0.0.0:", "//127.0.0.1:", "//localhost:"} {
			if strings.Contains(cfg.Host, lb) {
				cfg.Host = strings.Replace(cfg.Host, lb, "//host.docker.internal:", 1)
				cfg.TLSClientConfig.Insecure = true
				cfg.TLSClientConfig.CAData = nil
				cfg.TLSClientConfig.CAFile = ""
				break
			}
		}
	}
	return kubernetes.NewForConfig(cfg)
}

// flushLoop mirrors the docker flushLoop — pulls trailing multi-line
// entries through after a short idle.
func (c *K8sCollector) flushLoop(ctx context.Context) {
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

// watchClusterLoop runs the watch-reconnect loop for one cluster. K8s
// closes watches periodically (typically every 5-10 min); we just
// reconnect when the channel closes.
func (c *K8sCollector) watchClusterLoop(ctx context.Context, cluster string, cs *kubernetes.Clientset) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := c.watchOnce(ctx, cluster, cs); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("[logs/k8s] %s: watch error: %v — backing off %s", cluster, err, backoff)
			if !sleepCtx(ctx, backoff) {
				return
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

// watchOnce opens a Watch on pods cluster-wide for one cluster.
func (c *K8sCollector) watchOnce(ctx context.Context, cluster string, clientset *kubernetes.Clientset) error {
	// Seed with the current pod list so we don't miss pods that were
	// already running when we attached.
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := range pods.Items {
		if pods.Items[i].Status.Phase == corev1.PodRunning {
			c.attach(ctx, clientset, cluster, &pods.Items[i])
		}
	}

	w, err := clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
		ResourceVersion: pods.ResourceVersion,
	})
	if err != nil {
		return err
	}
	defer w.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-w.ResultChan():
			if !ok {
				// Server closed the watch; re-list-and-watch.
				return nil
			}
			pod, ok := ev.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			id := k8sPodID(cluster, pod)
			switch ev.Type {
			case watch.Added, watch.Modified:
				if pod.Status.Phase == corev1.PodRunning {
					c.attach(ctx, clientset, cluster, pod)
				} else if pod.Status.Phase == corev1.PodSucceeded ||
					pod.Status.Phase == corev1.PodFailed {
					c.detach(id)
				}
			case watch.Deleted:
				c.detach(id)
			}
		}
	}
}

// k8sPodID embeds the cluster name so the registry can hold streamers
// for same-named pods in different clusters (very common — `redis-0`
// in two namespaces of two different clusters would otherwise collide).
func k8sPodID(cluster string, p *corev1.Pod) string {
	return "k8s:" + cluster + "/" + p.Namespace + "/" + p.Name
}

func (c *K8sCollector) attach(parent context.Context, cs *kubernetes.Clientset, cluster string, pod *corev1.Pod) {
	id := k8sPodID(cluster, pod)
	_, ok := c.registry.Reserve(id)
	if !ok {
		c.registry.Touch(id)
		return
	}

	streamCtx, cancel := context.WithCancel(parent)
	meta := parser.SourceMeta{
		Platform:  logs.PlatformK8s,
		SourceID:  id,
		Cluster:   cluster,
		Pod:       pod.Name,
		Namespace: pod.Namespace,
		Service:   labelService(pod.Labels),
		Host:      pod.Spec.NodeName,
		Env:       c.env,
	}

	// Stream from the first container only — most pods are single-
	// container in practice, and additional containers can be reached
	// through the existing per-pod viewer.
	containerName := ""
	if len(pod.Spec.Containers) > 0 {
		containerName = pod.Spec.Containers[0].Name
	}
	if containerName == "" {
		c.registry.Release(id)
		cancel()
		return
	}

	go func() {
		defer cancel()
		c.stream(streamCtx, cs, pod.Namespace, pod.Name, containerName, meta)
		for _, e := range c.joiner.Forget(meta.SourceID) {
			c.submit.Submit(e)
		}
		c.registry.Release(id)
	}()
}

func (c *K8sCollector) detach(id string) {
	c.registry.Release(id)
}

// labelService picks a sensible "service" name from common K8s labels.
func labelService(labels map[string]string) string {
	for _, k := range []string{
		"app.kubernetes.io/name", "app", "k8s-app",
		"app.kubernetes.io/component",
	} {
		if v := labels[k]; v != "" {
			return v
		}
	}
	return ""
}

// stream is the per-pod read loop with reconnect-and-resume. SinceTime
// gives at-least-once continuity across API errors.
func (c *K8sCollector) stream(ctx context.Context, cs *kubernetes.Clientset, namespace, pod, container string, meta parser.SourceMeta) {
	var sinceTime *metav1.Time
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}
		opts := &corev1.PodLogOptions{
			Container:  container,
			Follow:     true,
			Timestamps: true,
		}
		if sinceTime != nil {
			opts.SinceTime = sinceTime
		}
		req := cs.CoreV1().Pods(namespace).GetLogs(pod, opts)
		stream, err := req.Stream(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("[logs/k8s] %s/%s: open failed: %v", namespace, pod, err)
			if !sleepCtx(ctx, backoff) {
				return
			}
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		lastTS := c.consumePodStream(ctx, stream, meta)
		_ = stream.Close()
		if !lastTS.IsZero() {
			ts := metav1.NewTime(lastTS)
			sinceTime = &ts
		}
		if !sleepCtx(ctx, time.Second) {
			return
		}
	}
}

// consumePodStream reads one open log stream until EOF / error. Returns
// the timestamp of the last successfully-parsed entry so the next
// reconnect can pick up cleanly.
func (c *K8sCollector) consumePodStream(ctx context.Context, stream interface {
	Read(p []byte) (int, error)
}, meta parser.SourceMeta) time.Time {
	scanner := bufio.NewScanner(readerFunc(stream.Read))
	scanner.Buffer(make([]byte, 4096), 128*1024)

	var last time.Time
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		c.registry.Touch(meta.SourceID)
		entry := parser.Parse(meta, line, time.Now())
		last = time.UnixMilli(entry.TS)
		for _, finished := range c.joiner.Feed(meta.SourceID, entry) {
			c.submit.Submit(finished)
		}
		if ctx.Err() != nil {
			return last
		}
	}
	return last
}

// readerFunc lets bufio.Scanner consume anything with a Read method
// without forcing the caller to expose io.Reader at the type level.
type readerFunc func(p []byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }
