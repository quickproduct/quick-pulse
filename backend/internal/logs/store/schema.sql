-- Logs module schema. Embedded via go:embed into sqlite.go and executed
-- alongside the existing tables in db.createTables().
--
-- Design notes:
--   * STRICT tables enforce types and reject silent coercions.
--   * Timestamps are INTEGER unix-millis so they sort and compare cheaply.
--   * Three indexes only — each additional one roughly doubles insert cost
--     and we're tight on CPU. These cover ~95% of UI queries.
--   * logs_fts is an external-content FTS5 table; we sync it from app code
--     in the same transaction as the INSERT (not via triggers) to keep CPU
--     predictable and avoid surprise lock contention.

CREATE TABLE IF NOT EXISTS logs (
    id        INTEGER PRIMARY KEY,
    ts        INTEGER NOT NULL,
    level     INTEGER NOT NULL,
    platform  INTEGER NOT NULL,
    source_id TEXT    NOT NULL,
    cluster   TEXT,
    container TEXT,
    pod       TEXT,
    namespace TEXT,
    service   TEXT,
    host      TEXT,
    env       TEXT,
    trace_id  TEXT,
    message   TEXT    NOT NULL,
    meta      TEXT
) STRICT;

CREATE INDEX IF NOT EXISTS idx_logs_ts         ON logs(ts DESC);
CREATE INDEX IF NOT EXISTS idx_logs_level_ts   ON logs(level, ts DESC);
CREATE INDEX IF NOT EXISTS idx_logs_source_ts  ON logs(source_id, ts DESC);
-- idx_logs_cluster_ts is created in code (sqlite.go::migrate) so it can be
-- applied after the migration ALTERs in an existing column. Putting it here
-- would fail on databases predating the `cluster` column.

CREATE VIRTUAL TABLE IF NOT EXISTS logs_fts USING fts5(
    message, container, pod, namespace, service,
    content='logs', content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

CREATE TABLE IF NOT EXISTS logs_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;
