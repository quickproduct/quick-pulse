package store

import "time"

// timeNowMS is split into its own file so tests can swap it without
// touching the rest of sqlite.go.
func timeNowMS() int64 { return time.Now().UnixMilli() }
