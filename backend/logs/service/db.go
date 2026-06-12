package service

import (
	"context"

	"quickpulse/backend/db"
)

// walCheckpoint forces a WAL checkpoint on the shared SQLite handle. We
// don't go through the store so we avoid the writeMu — checkpointing is
// safe to run concurrently with normal writers.
func walCheckpoint(ctx context.Context) (int64, error) {
	_, err := db.DB.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return 0, err
}
