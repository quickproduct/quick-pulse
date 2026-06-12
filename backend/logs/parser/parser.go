package parser

import (
	"strings"
	"time"

	"quickpulse/backend/logs"
)

// SourceMeta is the static metadata a collector knows about a source —
// passed once and stamped onto every line it produces. Keeps the parser
// stateless across calls.
type SourceMeta struct {
	Platform  logs.Platform
	SourceID  string
	Cluster   string
	Container string
	Pod       string
	Namespace string
	Service   string
	Host      string
	Env       string
}

// Parse converts one raw line into a fully-populated Entry. It performs
// JSON detection, level inference, timestamp extraction, and meta lifting
// — but never multi-line joining (that's the joiner's job).
//
// `received` is the time the collector observed the line. We use it as a
// fallback whenever the line itself doesn't carry a usable timestamp.
func Parse(meta SourceMeta, raw string, received time.Time) logs.Entry {
	// Strip the Docker/K8s timestamp prefix when present
	// (Docker: "2024-01-02T03:04:05.678Z stdout F message")
	// (K8s:    "2024-01-02T03:04:05.678Z message").
	ts := received
	line := raw
	if rest, t, ok := stripDockerTimestamp(line); ok {
		line = rest
		ts = t
	}

	e := logs.Entry{
		TS:        ts.UnixMilli(),
		Platform:  meta.Platform,
		SourceID:  meta.SourceID,
		Cluster:   meta.Cluster,
		Container: meta.Container,
		Pod:       meta.Pod,
		Namespace: meta.Namespace,
		Service:   meta.Service,
		Host:      meta.Host,
		Env:       meta.Env,
	}

	if j := ParseJSON(line); j.OK {
		e.Level = j.Level
		e.Message = j.Message
		e.TraceID = j.TraceID
		if j.Service != "" && e.Service == "" {
			e.Service = j.Service
		}
		e.Meta = j.Meta
		return e
	}

	e.Message = line
	e.Level = DetectLevel(line)
	return e
}

// stripDockerTimestamp removes the RFC3339Nano prefix that Docker / K8s
// prepend when Timestamps:true is set. Both platforms use the same format.
// Returns (remainder, parsed_ts, true) on a match.
func stripDockerTimestamp(line string) (string, time.Time, bool) {
	// Fast bail: the prefix is at least 20 chars and starts with a digit.
	if len(line) < 20 || line[0] < '0' || line[0] > '9' {
		return line, time.Time{}, false
	}
	sp := strings.IndexByte(line, ' ')
	if sp < 20 || sp > 40 {
		return line, time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, line[:sp])
	if err != nil {
		return line, time.Time{}, false
	}
	// Docker pipes through "stdout F" / "stderr P" stream markers — drop
	// them so the user sees clean text.
	rest := line[sp+1:]
	if (strings.HasPrefix(rest, "stdout ") || strings.HasPrefix(rest, "stderr ")) && len(rest) > 9 {
		// e.g. "stdout F message"
		if idx := strings.IndexByte(rest[7:], ' '); idx > 0 {
			rest = rest[7+idx+1:]
		}
	}
	return rest, t, true
}
