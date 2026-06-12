// Package store is the persistence seam for the logs module. The interface
// is intentionally narrow so we can later swap SQLite for Postgres, ClickHouse,
// or Loki without touching collectors, parsers, or HTTP handlers.
package store

import (
	"context"

	"quickpulse/backend/logs"
)

// Store is the only persistence interface the rest of the logs package knows
// about. All implementations must be safe for concurrent reads alongside a
// single writer (the batch-writer goroutine).
type Store interface {
	// Insert appends a batch of entries in a single transaction. The IDs in
	// the returned slice match the order of the input.
	Insert(ctx context.Context, batch []logs.Entry) ([]int64, error)

	// Query returns up to filter.Limit entries newest-first, plus a cursor
	// for the next page (empty when no more results).
	Query(ctx context.Context, f logs.Filter) ([]logs.Entry, string, error)

	// Get fetches a single entry by id.
	Get(ctx context.Context, id int64) (logs.Entry, error)

	// Sources returns the distinct values used to populate filter dropdowns.
	Sources(ctx context.Context) (logs.SourcesResponse, error)

	// Stats returns a time-bucketed histogram for the given filter.
	// bucketMs is the bucket size in milliseconds.
	Stats(ctx context.Context, f logs.Filter, bucketMs int64) ([]logs.StatsBucket, error)

	// Vacuum enforces retention: deletes rows older than retention_hours
	// and/or beyond max_size_mb. Returns the count of rows evicted.
	Vacuum(ctx context.Context, s logs.Settings) (int64, error)

	// Settings reads / writes persisted runtime config (retention,
	// sampling ratios). Defaults fill any missing keys.
	LoadSettings(ctx context.Context) (logs.Settings, error)
	SaveSettings(ctx context.Context, s logs.Settings) error

	// IncDropped bumps the dropped-counter shown in the UI banner.
	IncDropped(ctx context.Context, n int64) error
	DroppedCount(ctx context.Context) (int64, error)
}
