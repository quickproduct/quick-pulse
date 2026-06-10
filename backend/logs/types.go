// Package logs implements QuickPulse's centralized log ingestion, storage, and
// query layer. Collectors stream raw log lines from Docker containers and
// Kubernetes pods; the parser normalizes them; the sampler keeps the volume
// honest under the container's 64 MB / 0.25 CPU budget; the store persists
// into SQLite (FTS5-indexed) and the pub/sub broker fans out live tails to
// connected websockets.
package logs

import (
	"encoding/base64"
	"encoding/binary"
	"strconv"
)

// Level is a compact integer representation of log severity. Stored as INTEGER
// in SQLite to keep row size small — names are derived only at API boundaries.
type Level int

const (
	LevelDebug    Level = 0
	LevelInfo     Level = 1
	LevelWarn     Level = 2
	LevelError    Level = 3
	LevelCritical Level = 4
)

// LevelName returns the canonical uppercase name for a level.
func LevelName(l Level) string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelCritical:
		return "CRITICAL"
	}
	return "INFO"
}

// ParseLevel converts a user-supplied level name into the internal enum.
// Unknown values fall back to LevelInfo (the safest default).
func ParseLevel(s string) Level {
	switch s {
	case "DEBUG", "debug", "TRACE", "trace":
		return LevelDebug
	case "INFO", "info":
		return LevelInfo
	case "WARN", "warn", "WARNING", "warning":
		return LevelWarn
	case "ERROR", "error", "ERR", "err":
		return LevelError
	case "CRITICAL", "critical", "FATAL", "fatal":
		return LevelCritical
	}
	return LevelInfo
}

// Platform identifies where a log came from. Stored as INTEGER for cheap
// indexing and tiny row size.
type Platform int

const (
	PlatformDocker Platform = 0
	PlatformK8s    Platform = 1
)

func PlatformName(p Platform) string {
	if p == PlatformK8s {
		return "k8s"
	}
	return "docker"
}

func ParsePlatform(s string) (Platform, bool) {
	switch s {
	case "docker":
		return PlatformDocker, true
	case "k8s", "kubernetes":
		return PlatformK8s, true
	}
	return 0, false
}

// Entry is one parsed log line — the canonical shape that flows through the
// whole pipeline (collector → parser → sampler → store → pub/sub → API).
//
// `TS` is unix-millis so it serializes cheaply, sorts as INTEGER in SQLite,
// and survives JSON marshalling without loss. `Meta` is left as JSON-encoded
// bytes to avoid double serialization through the pipeline; the parser writes
// it once, the store inserts it as-is, the API streams it through.
type Entry struct {
	ID        int64    `json:"id"`
	TS        int64    `json:"ts"` // unix epoch ms
	Level     Level    `json:"level"`
	Platform  Platform `json:"platform"`
	SourceID  string   `json:"source_id"`
	Cluster   string   `json:"cluster,omitempty"` // kubeconfig context name for k8s; "docker" for local docker
	Container string   `json:"container,omitempty"`
	Pod       string   `json:"pod,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
	Service   string   `json:"service,omitempty"`
	Host      string   `json:"host,omitempty"`
	Env       string   `json:"env,omitempty"`
	TraceID   string   `json:"trace_id,omitempty"`
	Message   string   `json:"message"`
	Meta      string   `json:"meta,omitempty"` // raw JSON or empty
}

// PublicEntry is the wire shape sent to clients — keeps level/platform as
// readable strings so the UI doesn't need a translation table.
type PublicEntry struct {
	ID        int64  `json:"id"`
	TS        int64  `json:"ts"`
	Level     string `json:"level"`
	Platform  string `json:"platform"`
	SourceID  string `json:"source_id"`
	Cluster   string `json:"cluster,omitempty"`
	Container string `json:"container,omitempty"`
	Pod       string `json:"pod,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Service   string `json:"service,omitempty"`
	Host      string `json:"host,omitempty"`
	Env       string `json:"env,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	Message   string `json:"message"`
	Meta      string `json:"meta,omitempty"`
}

func (e Entry) ToPublic() PublicEntry {
	return PublicEntry{
		ID:        e.ID,
		TS:        e.TS,
		Level:     LevelName(e.Level),
		Platform:  PlatformName(e.Platform),
		SourceID:  e.SourceID,
		Cluster:   e.Cluster,
		Container: e.Container,
		Pod:       e.Pod,
		Namespace: e.Namespace,
		Service:   e.Service,
		Host:      e.Host,
		Env:       e.Env,
		TraceID:   e.TraceID,
		Message:   e.Message,
		Meta:      e.Meta,
	}
}

// Filter captures every dimension the search API supports. Empty slices /
// zero values mean "no filter on this dimension."
type Filter struct {
	Levels     []Level
	Platforms  []Platform
	Clusters   []string
	Containers []string
	Pods       []string
	Namespaces []string
	Services   []string
	Envs       []string
	From       int64 // unix ms, 0 = no lower bound
	To         int64 // unix ms, 0 = no upper bound
	Query      string
	Cursor     string
	Limit      int
}

// Cursor encodes (ts, id) as an opaque base64 string. The "(ts,id) <" tuple
// comparison gives a stable ordering even when many rows share the same ts.
type Cursor struct {
	TS int64
	ID int64
}

func EncodeCursor(c Cursor) string {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], uint64(c.TS))
	binary.BigEndian.PutUint64(buf[8:16], uint64(c.ID))
	return base64.URLEncoding.EncodeToString(buf[:])
}

func DecodeCursor(s string) (Cursor, bool) {
	if s == "" {
		return Cursor{}, false
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil || len(b) != 16 {
		return Cursor{}, false
	}
	return Cursor{
		TS: int64(binary.BigEndian.Uint64(b[0:8])),
		ID: int64(binary.BigEndian.Uint64(b[8:16])),
	}, true
}

// StatsBucket is one row in the histogram aggregation.
type StatsBucket struct {
	TS     int64          `json:"ts"`
	Count  int64          `json:"count"`
	Levels map[string]int `json:"by_level"`
}

// SourcesResponse drives the filter dropdowns. Counts are approximate
// (computed from DISTINCT on a bounded scan) and are mainly for sort hints.
type SourcesResponse struct {
	Containers []string `json:"containers"`
	Pods       []string `json:"pods"`
	Namespaces []string `json:"namespaces"`
	Services   []string `json:"services"`
	Envs       []string `json:"envs"`
	Clusters   []string `json:"clusters"`
	Platforms  []string `json:"platforms"`
	Dropped    int64    `json:"dropped"`
}

// Settings is the runtime-tunable config persisted in logs_meta. Defaults are
// applied in store.LoadSettings if any key is missing.
type Settings struct {
	RetentionHours int `json:"retention_hours"`
	MaxSizeMB      int `json:"max_size_mb"`
	SampleInfo     int `json:"sample_info"`  // 1 in N
	SampleDebug    int `json:"sample_debug"` // 1 in N
}

func DefaultSettings() Settings {
	return Settings{
		RetentionHours: 24,
		MaxSizeMB:      100,
		SampleInfo:     10,
		SampleDebug:    50,
	}
}

// itoa is here to avoid importing strconv in hot paths from other files in
// this package — keeps the import surface small.
func itoa(i int) string { return strconv.Itoa(i) }
