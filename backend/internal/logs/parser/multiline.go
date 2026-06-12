package parser

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"quickpulse/backend/internal/logs"
)

// stackFrameRe recognizes the most common continuation patterns:
//   - Java/Go: "    at com.foo.Bar(Baz.java:42)"
//   - Python:  '  File "foo.py", line 12, in bar'
//   - Generic: leading whitespace (handled separately below)
var stackFrameRe = regexp.MustCompile(`^(?:\s+at |\s*File "|Traceback|\s*[A-Z][A-Za-z0-9_.]+(?:Error|Exception): )`)

// MaxMessageBytes caps the size of a joined multi-line entry. Past this we
// truncate with a marker — full traces remain accessible from the per-source
// log viewer (Kubernetes/Container logs page), so no data is permanently lost.
const MaxMessageBytes = 16 * 1024

// MultilineJoiner buffers per-source partial entries until the next non-
// continuation line arrives (or a short idle timeout fires). Holding all
// partials in memory across thousands of streams would be expensive — but
// real Docker/K8s streams produce one stream per source, and we cap the
// active stream count in collector/registry.go, so the upper bound on
// memory here is `MAX_LOG_STREAMS * MaxMessageBytes` (~512 KB at default).
type MultilineJoiner struct {
	mu      sync.Mutex
	pending map[string]*pendingEntry
}

type pendingEntry struct {
	entry   logs.Entry
	updated time.Time
}

// NewMultilineJoiner constructs the per-source state holder.
func NewMultilineJoiner() *MultilineJoiner {
	return &MultilineJoiner{pending: map[string]*pendingEntry{}}
}

// Feed accepts the next parsed line for a given source and returns zero or
// more completed Entries ready for the ingest channel. The contract:
//
//   - If `next.Message` is a continuation of `sourceID`'s last entry, append
//     and return nothing.
//   - Otherwise, return the previously held entry (if any) and stash `next`.
//
// The caller must also periodically call Flush() to release entries that
// have gone idle.
func (m *MultilineJoiner) Feed(sourceID string, next logs.Entry) []logs.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()

	prev, hadPrev := m.pending[sourceID]

	if hadPrev && isContinuation(next.Message) {
		// Append, truncate if needed, keep waiting.
		if len(prev.entry.Message)+1+len(next.Message) > MaxMessageBytes {
			remaining := MaxMessageBytes - len(prev.entry.Message)
			if remaining > 16 {
				prev.entry.Message += "\n" + next.Message[:remaining-len("\n…[truncated]")] + "\n…[truncated]"
			}
		} else {
			prev.entry.Message += "\n" + next.Message
		}
		prev.updated = time.Now()
		return nil
	}

	// Not a continuation: flush previous, stash next.
	var out []logs.Entry
	if hadPrev {
		out = append(out, prev.entry)
	}
	m.pending[sourceID] = &pendingEntry{entry: next, updated: time.Now()}
	return out
}

// Flush releases any entry older than `idle`. Called periodically from the
// ingest tick so we don't leave half-finished entries floating forever.
func (m *MultilineJoiner) Flush(idle time.Duration) []logs.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-idle)
	var out []logs.Entry
	for k, p := range m.pending {
		if p.updated.Before(cutoff) {
			out = append(out, p.entry)
			delete(m.pending, k)
		}
	}
	return out
}

// FlushAll drains every pending entry — used at shutdown.
func (m *MultilineJoiner) FlushAll() []logs.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]logs.Entry, 0, len(m.pending))
	for _, p := range m.pending {
		out = append(out, p.entry)
	}
	m.pending = map[string]*pendingEntry{}
	return out
}

// Forget drops state for a source that's gone away (container died, pod
// deleted). Releases the trailing entry first so we don't lose its tail.
func (m *MultilineJoiner) Forget(sourceID string) []logs.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pending[sourceID]
	if !ok {
		return nil
	}
	delete(m.pending, sourceID)
	return []logs.Entry{p.entry}
}

// isContinuation returns true when `line` should attach to the previous
// entry rather than start a new one.
func isContinuation(line string) bool {
	if line == "" {
		return false
	}
	if line[0] == ' ' || line[0] == '\t' {
		return true
	}
	return stackFrameRe.MatchString(line)
}

// _ keeps the strings import alive even when we trim future stack patterns
// out — leave it as a no-op rather than churning imports.
var _ = strings.TrimSpace
