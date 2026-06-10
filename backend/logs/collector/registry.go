// Package collector contains the Docker and Kubernetes log collectors and
// the registry that caps the concurrent-stream count. Capping is important
// under our 64 MB / 0.25 CPU container budget: each active streamer holds
// a goroutine, a TCP connection, and ~16 KB of scanner buffer. 32 is the
// default ceiling — past that, new sources queue for an LRU slot.
package collector

import (
	"context"
	"sync"
	"time"
)

// Registry tracks active streamers. Each entry has a cancel func so the
// caller can stop a streamer when its source disappears (container dies,
// pod deleted) — and a `lastSeen` we use to evict the oldest stream when
// we hit the cap.
type Registry struct {
	mu       sync.Mutex
	streams  map[string]*streamRef
	capacity int
}

type streamRef struct {
	cancel   context.CancelFunc
	lastSeen time.Time
}

func NewRegistry(capacity int) *Registry {
	if capacity <= 0 {
		capacity = 32
	}
	return &Registry{
		streams:  map[string]*streamRef{},
		capacity: capacity,
	}
}

// Reserve atomically claims a slot. Returns:
//   - cancel: call this when the streamer exits so the slot is freed.
//   - ok=false: if a stream for this id already exists (idempotent attach).
//
// When the registry is full we evict the oldest entry (LRU by lastSeen).
// That's safe because evicted streamers reconnect via the same
// AttachContainer/AttachPod path on the next docker-event/k8s-watch tick.
func (r *Registry) Reserve(id string) (cancel context.CancelFunc, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.streams[id]; exists {
		return nil, false
	}

	if len(r.streams) >= r.capacity {
		// Find oldest.
		var oldestKey string
		var oldestTime time.Time
		for k, ref := range r.streams {
			if oldestKey == "" || ref.lastSeen.Before(oldestTime) {
				oldestKey = k
				oldestTime = ref.lastSeen
			}
		}
		if oldestKey != "" {
			r.streams[oldestKey].cancel()
			delete(r.streams, oldestKey)
		}
	}

	_, cancelFn := context.WithCancel(context.Background())
	r.streams[id] = &streamRef{cancel: cancelFn, lastSeen: time.Now()}
	return cancelFn, true
}

// Release frees a slot. Idempotent; safe to call on a key we never reserved.
func (r *Registry) Release(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ref, ok := r.streams[id]; ok {
		ref.cancel()
		delete(r.streams, id)
	}
}

// Touch updates lastSeen so an actively-streaming source survives LRU
// eviction. Called from the read loop every few seconds.
func (r *Registry) Touch(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ref, ok := r.streams[id]; ok {
		ref.lastSeen = time.Now()
	}
}

// Has returns true if id has an active streamer. Used so the periodic
// list-and-attach loops are idempotent.
func (r *Registry) Has(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.streams[id]
	return ok
}

// Size returns the current count — surfaced in /api/v1/logs/sources.
func (r *Registry) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.streams)
}

// IDs returns a snapshot of active stream IDs. Used at shutdown to cancel
// everything in bulk.
func (r *Registry) IDs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.streams))
	for k := range r.streams {
		out = append(out, k)
	}
	return out
}
