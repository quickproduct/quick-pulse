package parser

import "strconv"

// sprintfFloat is a single-use helper so json.go doesn't have to import fmt.
// strconv is already in our binary via the workers; cost is zero.
func sprintfFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
