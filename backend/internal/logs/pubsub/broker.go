// Package pubsub fans out committed log batches to live WebSocket
// subscribers. It's intentionally simpler than ws/manager.go because
// every subscriber has its own filter — we evaluate the filter once per
// entry per subscriber rather than maintaining one channel per filter.
//
// With per-connection ring-buffered delivery and a hard cap on subscriber
// count (LOGS_MAX_SUBSCRIBERS, default 16), the broker's memory ceiling
// is bounded at ~16 × 500 entries ≈ 8 MB worst case.
package pubsub

import (
	"strings"
	"sync"
	"sync/atomic"

	"quickpulse/backend/internal/logs"
)

// SubscriberFilter is a subset of logs.Filter the broker can evaluate
// in-memory (no DB roundtrip). Strings are pre-sorted so we can binary-
// search but the dataset is tiny so we do linear scans for simplicity.
type SubscriberFilter struct {
	Levels     []logs.Level
	Platforms  []logs.Platform
	Clusters   []string
	Containers []string
	Pods       []string
	Namespaces []string
	Services   []string
	Envs       []string
	Query      string // case-insensitive substring match
}

// Matches returns true when `e` should be delivered to a subscriber holding
// this filter. Empty slice == "no filter on this dimension".
func (f SubscriberFilter) Matches(e logs.Entry) bool {
	if !levelIn(f.Levels, e.Level) {
		return false
	}
	if !platformIn(f.Platforms, e.Platform) {
		return false
	}
	if !stringIn(f.Clusters, e.Cluster) {
		return false
	}
	if !stringIn(f.Containers, e.Container) {
		return false
	}
	if !stringIn(f.Pods, e.Pod) {
		return false
	}
	if !stringIn(f.Namespaces, e.Namespace) {
		return false
	}
	if !stringIn(f.Services, e.Service) {
		return false
	}
	if !stringIn(f.Envs, e.Env) {
		return false
	}
	if f.Query != "" {
		if !strings.Contains(strings.ToLower(e.Message), strings.ToLower(f.Query)) {
			return false
		}
	}
	return true
}

func levelIn(ls []logs.Level, v logs.Level) bool {
	if len(ls) == 0 {
		return true
	}
	for _, l := range ls {
		if l == v {
			return true
		}
	}
	return false
}

func platformIn(ps []logs.Platform, v logs.Platform) bool {
	if len(ps) == 0 {
		return true
	}
	for _, p := range ps {
		if p == v {
			return true
		}
	}
	return false
}

func stringIn(ss []string, v string) bool {
	if len(ss) == 0 {
		return true
	}
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

// Subscriber represents one live WebSocket. The broker writes batches into
// `Out` via the non-blocking sendBatch path; the WS handler reads from `Out`
// and serializes to the wire.
type Subscriber struct {
	ID      uint64
	Out     chan []logs.PublicEntry
	Dropped int64 // total entries dropped to keep this subscriber's buffer bounded

	filterMu sync.RWMutex
	filter   SubscriberFilter
	paused   atomic.Bool
}

// Filter returns the current filter snapshot. Cheap to call from the hot
// publish path because we keep it inline-able.
func (s *Subscriber) Filter() SubscriberFilter {
	s.filterMu.RLock()
	defer s.filterMu.RUnlock()
	return s.filter
}

func (s *Subscriber) SetFilter(f SubscriberFilter) {
	s.filterMu.Lock()
	s.filter = f
	s.filterMu.Unlock()
}

func (s *Subscriber) Pause()  { s.paused.Store(true) }
func (s *Subscriber) Resume() { s.paused.Store(false) }
func (s *Subscriber) Paused() bool {
	return s.paused.Load()
}

// Broker holds all current subscribers. Publish iterates them with an
// RLock so subscribe/unsubscribe never blocks the writer.
type Broker struct {
	mu     sync.RWMutex
	subs   map[uint64]*Subscriber
	nextID uint64
	maxSub int // 0 = unlimited; in practice we cap at 16
}

func NewBroker(maxSubscribers int) *Broker {
	return &Broker{
		subs:   map[uint64]*Subscriber{},
		maxSub: maxSubscribers,
	}
}

// Subscribe registers a new subscriber. Returns nil when the broker has
// reached the maxSubscribers cap, so the caller can close the WS with a
// 503-style error.
func (b *Broker) Subscribe(initial SubscriberFilter, bufferSize int) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.maxSub > 0 && len(b.subs) >= b.maxSub {
		return nil
	}
	if bufferSize <= 0 {
		bufferSize = 64
	}
	b.nextID++
	s := &Subscriber{
		ID:     b.nextID,
		Out:    make(chan []logs.PublicEntry, bufferSize),
		filter: initial,
	}
	b.subs[s.ID] = s
	return s
}

func (b *Broker) Unsubscribe(s *Subscriber) {
	b.mu.Lock()
	delete(b.subs, s.ID)
	b.mu.Unlock()
	close(s.Out)
}

// Publish fans `batch` out to every subscriber whose filter matches at
// least one entry. We materialize a per-subscriber slice only when at
// least one entry matches — avoids per-line allocations on quiet streams.
//
// Backpressure: each Out channel is bounded; if a subscriber's reader is
// slow we drop the *oldest queued batch* (popping one off Out, pushing
// the new one) and bump Dropped. This means we never block the publisher
// goroutine — critical because Publish is called inline from the ingest
// path.
func (b *Broker) Publish(batch []logs.Entry) {
	b.mu.RLock()
	subs := make([]*Subscriber, 0, len(b.subs))
	for _, s := range b.subs {
		subs = append(subs, s)
	}
	b.mu.RUnlock()

	if len(subs) == 0 {
		return
	}

	for _, s := range subs {
		if s.Paused() {
			continue
		}
		f := s.Filter()
		var out []logs.PublicEntry
		for _, e := range batch {
			if f.Matches(e) {
				out = append(out, e.ToPublic())
			}
		}
		if len(out) == 0 {
			continue
		}
		select {
		case s.Out <- out:
		default:
			// Buffer full — drop oldest, push newest.
			select {
			case <-s.Out:
				atomic.AddInt64(&s.Dropped, int64(len(out)))
			default:
			}
			select {
			case s.Out <- out:
			default:
				// Still full (race) — give up; the reader is hopelessly
				// behind and will get a "dropped" notice next round.
				atomic.AddInt64(&s.Dropped, int64(len(out)))
			}
		}
	}
}

// Count returns the number of active subscribers — useful for `/health`
// and the sources endpoint.
func (b *Broker) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}
