package service

import (
	"context"
	"hash/fnv"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"quickpulse/backend/logs"
)

// Ingest owns the bounded ingest channel and the single batch-writer
// goroutine. Producers (collectors) call Submit; the writer flushes to the
// store on a timer and publishes committed batches to the broker.
//
// This is the only place in the module that mutates the database — every
// other code path is a reader. That invariant is what lets SQLite stay at
// MaxOpenConns=1 without contention.
type Ingest struct {
	store     EntryStore
	publisher BatchPublisher
	settings  *atomic.Value // holds Settings

	ch      chan logs.Entry
	dropped int64

	batchSize    int
	batchTimeout time.Duration

	wg       sync.WaitGroup
	shutdown chan struct{}
}

// EntryStore narrows the store interface to just what the ingest path
// needs — keeps tests light and decouples this file from the store package.
type EntryStore interface {
	Insert(ctx context.Context, batch []logs.Entry) ([]int64, error)
	IncDropped(ctx context.Context, n int64) error
}

// BatchPublisher is the broker contract. Defined here as an interface so we
// can swap in a no-op for tests.
type BatchPublisher interface {
	Publish(batch []logs.Entry)
}

// NewIngest returns a started ingest pipeline. Cancel ctx to drain & stop.
//
// `bufferSize` caps the in-flight entries (default 1024 ≈ 2 MB RSS).
// `batchSize` and `batchTimeout` control the flush cadence.
func NewIngest(
	ctx context.Context,
	store EntryStore,
	publisher BatchPublisher,
	settings logs.Settings,
	bufferSize, batchSize int,
	batchTimeout time.Duration,
) *Ingest {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	if batchSize <= 0 {
		batchSize = 200
	}
	if batchTimeout <= 0 {
		batchTimeout = 250 * time.Millisecond
	}

	st := &atomic.Value{}
	st.Store(settings)

	i := &Ingest{
		store:        store,
		publisher:    publisher,
		settings:     st,
		ch:           make(chan logs.Entry, bufferSize),
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		shutdown:     make(chan struct{}),
	}
	i.wg.Add(1)
	go i.run(ctx)
	return i
}

// UpdateSettings swaps the live config (used by the settings endpoint to
// apply new sample ratios without a restart). The change is picked up on
// the next call to Submit.
func (i *Ingest) UpdateSettings(s logs.Settings) {
	i.settings.Store(s)
}

// Submit applies sampling and enqueues an entry. Never blocks: if the
// channel is full, the entry is dropped and the dropped counter ticked.
// Producers can therefore call this from hot loops without backpressure.
func (i *Ingest) Submit(e logs.Entry) {
	s := i.settings.Load().(logs.Settings)
	if !i.sample(e, s) {
		return
	}
	select {
	case i.ch <- e:
	default:
		atomic.AddInt64(&i.dropped, 1)
	}
}

// sample applies the level-based ratio. Decision is deterministic per
// (source, level) so the same source consistently keeps or drops the same
// fraction — gives fair coverage across all sources.
func (i *Ingest) sample(e logs.Entry, s logs.Settings) bool {
	switch e.Level {
	case logs.LevelWarn, logs.LevelError, logs.LevelCritical:
		return true
	case logs.LevelInfo:
		return s.SampleInfo <= 1 || hashMod(e.SourceID, s.SampleInfo) == 0
	case logs.LevelDebug:
		return s.SampleDebug <= 1 || hashMod(e.SourceID, s.SampleDebug) == 0
	}
	return true
}

// hashMod returns a stable hash(sourceID) % n. fnv32 is the cheapest
// non-cryptographic hash in the stdlib.
func hashMod(s string, n int) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	if n <= 0 {
		return 0
	}
	return h.Sum32() % uint32(n)
}

// run is the batch-writer goroutine. It accumulates entries until either
// batchSize is reached or batchTimeout elapses, then flushes in one tx.
func (i *Ingest) run(ctx context.Context) {
	defer i.wg.Done()
	defer close(i.shutdown)

	buf := make([]logs.Entry, 0, i.batchSize)
	ticker := time.NewTicker(i.batchTimeout)
	defer ticker.Stop()

	// Persisted dropped-counter only needs to update every ~10 s — saves
	// a write per batch when no drops happened.
	var lastDroppedSync int64
	droppedTicker := time.NewTicker(10 * time.Second)
	defer droppedTicker.Stop()

	flush := func(reason string) {
		if len(buf) == 0 {
			return
		}
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ids, err := i.store.Insert(writeCtx, buf)
		cancel()
		if err != nil {
			log.Printf("[logs/ingest] flush failed (%s, n=%d): %v", reason, len(buf), err)
			// Drop the batch rather than retry forever — keeps the
			// pipeline live. Restart-loop will reconnect collectors,
			// dropping is observable via the counter.
			atomic.AddInt64(&i.dropped, int64(len(buf)))
			buf = buf[:0]
			return
		}
		for j := range buf {
			buf[j].ID = ids[j]
		}
		if i.publisher != nil {
			i.publisher.Publish(buf)
		}
		buf = buf[:0]
	}

	for {
		select {
		case <-ctx.Done():
			// Drain whatever is still in the channel, then exit.
			for {
				select {
				case e := <-i.ch:
					buf = append(buf, e)
					if len(buf) >= i.batchSize {
						flush("shutdown-drain")
					}
				default:
					flush("shutdown")
					return
				}
			}
		case e := <-i.ch:
			buf = append(buf, e)
			if len(buf) >= i.batchSize {
				flush("size")
			}
		case <-ticker.C:
			flush("tick")
		case <-droppedTicker.C:
			cur := atomic.LoadInt64(&i.dropped)
			if cur != lastDroppedSync {
				delta := cur - lastDroppedSync
				lastDroppedSync = cur
				dctx, dcancel := context.WithTimeout(context.Background(), 2*time.Second)
				if err := i.store.IncDropped(dctx, delta); err != nil {
					log.Printf("[logs/ingest] dropped-counter persist failed: %v", err)
				}
				dcancel()
			}
		}
	}
}

// Stop blocks until the batch-writer goroutine has drained and exited.
// Called from main during graceful shutdown (not currently wired but cheap
// to support).
func (i *Ingest) Stop() {
	<-i.shutdown
	i.wg.Wait()
}

// Dropped returns the in-memory dropped counter — used by ad-hoc debug
// endpoints. The persisted counter lives in logs_meta.
func (i *Ingest) Dropped() int64 {
	return atomic.LoadInt64(&i.dropped)
}
