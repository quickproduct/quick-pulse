// Package parser turns a raw line of text (plus the source's known metadata)
// into a normalized logs.Entry. It is intentionally pure and fast: no I/O,
// no allocations beyond what the input forces. The pipeline calls it from a
// hot loop in each per-source goroutine, so every microsecond matters under
// the 0.25 CPU container budget.
package parser

import (
	"regexp"

	"quickpulse/backend/logs"
)

// levelRe matches the first occurrence of a canonical level word anywhere
// in the line. Order in the alternation is unimportant for correctness
// (regex picks longest at the first match), but we keep severity-descending
// for readability.
var levelRe = regexp.MustCompile(`(?i)\b(CRITICAL|FATAL|ERROR|ERR|WARN(?:ING)?|INFO|DEBUG|TRACE)\b`)

// DetectLevel scans a line for a level keyword. Returns LevelInfo when no
// keyword is found — that's the safest fallback because INFO-default is the
// convention in container ecosystems (Docker logs, stdout from CLI tools).
//
// Important: we use FindIndex rather than FindString to avoid an allocation
// for the matched substring.
func DetectLevel(line string) logs.Level {
	loc := levelRe.FindStringIndex(line)
	if loc == nil {
		return logs.LevelInfo
	}
	// Compare the matched bytes case-insensitively without allocating.
	m := line[loc[0]:loc[1]]
	switch m[0] {
	case 'C', 'c':
		return logs.LevelCritical
	case 'F', 'f':
		return logs.LevelCritical
	case 'E', 'e':
		return logs.LevelError
	case 'W', 'w':
		return logs.LevelWarn
	case 'D', 'd':
		return logs.LevelDebug
	case 'T', 't':
		return logs.LevelDebug
	}
	return logs.LevelInfo
}
