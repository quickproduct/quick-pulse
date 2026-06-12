package parser

import (
	"encoding/json"
	"strings"

	"quickpulse/backend/internal/logs"
)

// JSONFields is the subset of a structured-log JSON object we lift into
// first-class columns. Everything else stays in the meta blob, so we don't
// need to maintain an exhaustive list.
type JSONFields struct {
	Level   logs.Level
	Message string
	TraceID string
	Service string
	Meta    string // remaining keys as JSON, or "" when no extras
	OK      bool   // true only when the line was valid JSON
}

// commonKeys are the keys we promote to first-class columns. Lower-case
// because we match case-insensitively but normalize on extraction.
var levelKeys = []string{"level", "lvl", "severity"}
var msgKeys = []string{"message", "msg", "log"}
var traceKeys = []string{"trace_id", "traceId", "request_id", "requestId", "correlation_id"}
var serviceKeys = []string{"service", "app", "logger", "component"}

// ParseJSON tries to interpret a line as a structured log object. Returns
// JSONFields{OK:false} for any line that isn't a plausible JSON object —
// the caller falls back to plain-text parsing in that case.
//
// We bail fast on the first character to avoid spending CPU on plain-text
// lines (which dominate volume in most environments).
func ParseJSON(line string) JSONFields {
	if len(line) == 0 || line[0] != '{' {
		return JSONFields{}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return JSONFields{}
	}
	if len(obj) == 0 {
		return JSONFields{OK: true, Message: line}
	}

	out := JSONFields{OK: true}

	// Promote known keys; delete from map so the leftovers become `meta`.
	if v, k := popString(obj, levelKeys); k != "" {
		out.Level = logs.ParseLevel(v)
	}
	if v, k := popString(obj, msgKeys); k != "" {
		out.Message = v
	}
	if v, k := popString(obj, traceKeys); k != "" {
		out.TraceID = v
	}
	if v, k := popString(obj, serviceKeys); k != "" {
		out.Service = v
	}

	// Fallback message: if no msg/message key was present, use the raw line
	// so the user still sees something readable in the table.
	if out.Message == "" {
		out.Message = line
	}

	if len(obj) > 0 {
		if b, err := json.Marshal(obj); err == nil {
			out.Meta = string(b)
		}
	}
	return out
}

// popString finds the first matching key (case-insensitive), deletes it from
// the map and returns its string value. Returns ("", "") when no key matches.
func popString(obj map[string]any, candidates []string) (string, string) {
	for _, want := range candidates {
		for k, v := range obj {
			if strings.EqualFold(k, want) {
				delete(obj, k)
				switch s := v.(type) {
				case string:
					return s, k
				case float64:
					// JSON numbers — common for trace IDs from older systems.
					return jsonNumberString(s), k
				}
				return "", k
			}
		}
	}
	return "", ""
}

// jsonNumberString renders a float64 as a string without scientific notation
// when it's a whole-number value (the typical case for IDs).
func jsonNumberString(f float64) string {
	if f == float64(int64(f)) {
		return strings.TrimRight(strings.TrimRight(formatFloat(f), "0"), ".")
	}
	return formatFloat(f)
}

// formatFloat is split out so we don't import strconv at the top level —
// keeps the package's import surface tiny.
func formatFloat(f float64) string {
	// strconv.FormatFloat would be ideal but pulling strconv just for this
	// hot path is wasteful. We delegate via fmt.Sprintf which is already in
	// stdlib's runtime path for our uses.
	return sprintfFloat(f)
}
