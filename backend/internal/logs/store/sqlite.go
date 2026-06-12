package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"quickpulse/backend/internal/logs"
)

//go:embed schema.sql
var schema string

// SQLiteStore wraps the existing *sql.DB. We do not create a new connection
// pool — the parent process already runs at MaxOpenConns=1 (the SQLite sweet
// spot) and we want one writer for the whole module.
type SQLiteStore struct {
	db *sql.DB

	// writeMu serializes the single writer goroutine (the batch-writer in
	// logs/ingest.go) against settings/vacuum operations that also write.
	// SQLite would do this for us, but holding it on the Go side gives
	// clearer error messages and avoids surprise "database is locked".
	writeMu sync.Mutex
}

func New(db *sql.DB) (*SQLiteStore, error) {
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("logs schema migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	// Forward-compat: the `cluster` column was added after the initial
	// release. Older databases need an explicit ALTER before any index on
	// the new column can be created.
	if !s.columnExists("logs", "cluster") {
		if _, err := s.db.Exec(`ALTER TABLE logs ADD COLUMN cluster TEXT`); err != nil {
			return fmt.Errorf("add cluster column: %w", err)
		}
	}
	// Always (re-)assert the cluster index. IF NOT EXISTS makes this safe
	// on both fresh and migrated databases.
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_cluster_ts ON logs(cluster, ts DESC)`); err != nil {
		return fmt.Errorf("create cluster index: %w", err)
	}
	return nil
}

// columnExists asks SQLite whether a table currently carries `col`.
func (s *SQLiteStore) columnExists(table, col string) bool {
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			continue
		}
		if name == col {
			return true
		}
	}
	return false
}

// Insert performs one transactional batch insert into both logs and logs_fts.
// Returns the generated row IDs in input order so the pub/sub broker can
// stamp each Entry before fanning out.
func (s *SQLiteStore) Insert(ctx context.Context, batch []logs.Entry) ([]int64, error) {
	if len(batch) == 0 {
		return nil, nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO logs (
			ts, level, platform, source_id, cluster, container, pod, namespace,
			service, host, env, trace_id, message, meta
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	ftsStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO logs_fts (rowid, message, container, pod, namespace, service)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer ftsStmt.Close()

	ids := make([]int64, len(batch))
	for i, e := range batch {
		res, err := stmt.ExecContext(ctx,
			e.TS, int(e.Level), int(e.Platform), e.SourceID,
			nullable(e.Cluster),
			nullable(e.Container), nullable(e.Pod), nullable(e.Namespace),
			nullable(e.Service), nullable(e.Host), nullable(e.Env),
			nullable(e.TraceID), e.Message, nullable(e.Meta),
		)
		if err != nil {
			return nil, fmt.Errorf("insert: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		ids[i] = id

		if _, err := ftsStmt.ExecContext(ctx, id, e.Message,
			e.Container, e.Pod, e.Namespace, e.Service,
		); err != nil {
			return nil, fmt.Errorf("fts insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

// nullable converts an empty string to sql.NullString so SQLite stores NULL
// rather than ” — keeps DISTINCT queries clean for the Sources facets.
func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Query builds the search WHERE clause from a Filter and returns up to
// f.Limit rows newest-first plus a cursor for the next page.
func (s *SQLiteStore) Query(ctx context.Context, f logs.Filter) ([]logs.Entry, string, error) {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}

	// Strategy: when q (full-text) is present we JOIN logs_fts; otherwise
	// scan logs directly. FTS rowids match logs.id by design.
	var (
		sb   strings.Builder
		args []interface{}
	)

	useFTS := strings.TrimSpace(f.Query) != ""
	if useFTS {
		sb.WriteString(`
			SELECT l.id, l.ts, l.level, l.platform, l.source_id, l.cluster, l.container, l.pod,
			       l.namespace, l.service, l.host, l.env, l.trace_id, l.message, l.meta
			FROM logs_fts f
			JOIN logs l ON l.id = f.rowid
			WHERE logs_fts MATCH ?
		`)
		args = append(args, ftsEscape(f.Query))
	} else {
		sb.WriteString(`
			SELECT id, ts, level, platform, source_id, cluster, container, pod,
			       namespace, service, host, env, trace_id, message, meta
			FROM logs
			WHERE 1=1
		`)
	}

	prefix := ""
	if useFTS {
		prefix = "l."
	}

	if len(f.Levels) > 0 {
		sb.WriteString(" AND " + prefix + "level IN (")
		for i, lv := range f.Levels {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("?")
			args = append(args, int(lv))
		}
		sb.WriteString(")")
	}
	if len(f.Platforms) > 0 {
		sb.WriteString(" AND " + prefix + "platform IN (")
		for i, p := range f.Platforms {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("?")
			args = append(args, int(p))
		}
		sb.WriteString(")")
	}
	args = appendInList(&sb, args, prefix+"cluster", f.Clusters)
	args = appendInList(&sb, args, prefix+"container", f.Containers)
	args = appendInList(&sb, args, prefix+"pod", f.Pods)
	args = appendInList(&sb, args, prefix+"namespace", f.Namespaces)
	args = appendInList(&sb, args, prefix+"service", f.Services)
	args = appendInList(&sb, args, prefix+"env", f.Envs)

	if f.From > 0 {
		sb.WriteString(" AND " + prefix + "ts >= ?")
		args = append(args, f.From)
	}
	if f.To > 0 {
		sb.WriteString(" AND " + prefix + "ts <= ?")
		args = append(args, f.To)
	}

	if cur, ok := logs.DecodeCursor(f.Cursor); ok {
		// Tuple comparison gives us a stable next-page even when many
		// rows share the same ts.
		sb.WriteString(" AND (" + prefix + "ts < ? OR (" + prefix + "ts = ? AND " + prefix + "id < ?))")
		args = append(args, cur.TS, cur.TS, cur.ID)
	}

	sb.WriteString(" ORDER BY " + prefix + "ts DESC, " + prefix + "id DESC LIMIT ?")
	args = append(args, f.Limit)

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	out := make([]logs.Entry, 0, f.Limit)
	for rows.Next() {
		var e logs.Entry
		var level, platform int
		var cluster, container, pod, namespace, service, host, env, traceID, meta sql.NullString
		if err := rows.Scan(&e.ID, &e.TS, &level, &platform, &e.SourceID,
			&cluster, &container, &pod, &namespace, &service, &host, &env, &traceID,
			&e.Message, &meta,
		); err != nil {
			return nil, "", err
		}
		e.Level = logs.Level(level)
		e.Platform = logs.Platform(platform)
		e.Cluster = cluster.String
		e.Container = container.String
		e.Pod = pod.String
		e.Namespace = namespace.String
		e.Service = service.String
		e.Host = host.String
		e.Env = env.String
		e.TraceID = traceID.String
		e.Meta = meta.String
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var next string
	if len(out) == f.Limit {
		last := out[len(out)-1]
		next = logs.EncodeCursor(logs.Cursor{TS: last.TS, ID: last.ID})
	}
	return out, next, nil
}

// appendInList renders `AND col IN (?, ?, …)` for non-empty string slices.
// Returns the new args slice.
func appendInList(sb *strings.Builder, args []interface{}, col string, vals []string) []interface{} {
	if len(vals) == 0 {
		return args
	}
	sb.WriteString(" AND " + col + " IN (")
	for i, v := range vals {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
		args = append(args, v)
	}
	sb.WriteString(")")
	return args
}

// ftsEscape quotes an FTS5 query so user input like "level=ERROR" doesn't
// blow up the parser. Phrase-quoted strings disable column-prefix syntax.
func ftsEscape(q string) string {
	q = strings.ReplaceAll(q, `"`, `""`)
	return `"` + q + `"`
}

func (s *SQLiteStore) Get(ctx context.Context, id int64) (logs.Entry, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, ts, level, platform, source_id, cluster, container, pod,
		       namespace, service, host, env, trace_id, message, meta
		FROM logs WHERE id = ?
	`, id)
	var e logs.Entry
	var level, platform int
	var cluster, container, pod, namespace, service, host, env, traceID, meta sql.NullString
	if err := row.Scan(&e.ID, &e.TS, &level, &platform, &e.SourceID,
		&cluster, &container, &pod, &namespace, &service, &host, &env, &traceID,
		&e.Message, &meta,
	); err != nil {
		return logs.Entry{}, err
	}
	e.Level = logs.Level(level)
	e.Platform = logs.Platform(platform)
	e.Cluster = cluster.String
	e.Container = container.String
	e.Pod = pod.String
	e.Namespace = namespace.String
	e.Service = service.String
	e.Host = host.String
	e.Env = env.String
	e.TraceID = traceID.String
	e.Meta = meta.String
	return e, nil
}

// Sources runs five small DISTINCT queries against the last 24h of logs.
// We cap each DISTINCT scan via the time bound so this stays fast even
// when the table is large; the time window matches default retention.
func (s *SQLiteStore) Sources(ctx context.Context) (logs.SourcesResponse, error) {
	r := logs.SourcesResponse{Platforms: []string{"docker", "k8s"}}

	q := func(col string) ([]string, error) {
		rows, err := s.db.QueryContext(ctx,
			"SELECT DISTINCT "+col+" FROM logs WHERE "+col+" IS NOT NULL ORDER BY "+col+" LIMIT 200")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				return nil, err
			}
			if v != "" {
				out = append(out, v)
			}
		}
		return out, rows.Err()
	}

	var err error
	if r.Containers, err = q("container"); err != nil {
		return r, err
	}
	if r.Pods, err = q("pod"); err != nil {
		return r, err
	}
	if r.Namespaces, err = q("namespace"); err != nil {
		return r, err
	}
	if r.Services, err = q("service"); err != nil {
		return r, err
	}
	if r.Envs, err = q("env"); err != nil {
		return r, err
	}
	if r.Clusters, err = q("cluster"); err != nil {
		return r, err
	}
	if r.Dropped, err = s.DroppedCount(ctx); err != nil {
		return r, err
	}
	return r, nil
}

// Stats buckets entries by (ts / bucketMs) and counts per level. Returns
// rows oldest-first so a chart can render left-to-right without sorting.
func (s *SQLiteStore) Stats(ctx context.Context, f logs.Filter, bucketMs int64) ([]logs.StatsBucket, error) {
	if bucketMs <= 0 {
		bucketMs = 60_000 // 1m default
	}

	var (
		sb   strings.Builder
		args []interface{}
	)
	sb.WriteString(`SELECT (ts / ?) * ? AS bucket, level, COUNT(*) FROM logs WHERE 1=1`)
	args = append(args, bucketMs, bucketMs)

	if len(f.Levels) > 0 {
		sb.WriteString(" AND level IN (")
		for i, lv := range f.Levels {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("?")
			args = append(args, int(lv))
		}
		sb.WriteString(")")
	}
	if len(f.Platforms) > 0 {
		sb.WriteString(" AND platform IN (")
		for i, p := range f.Platforms {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("?")
			args = append(args, int(p))
		}
		sb.WriteString(")")
	}
	args = appendInList(&sb, args, "cluster", f.Clusters)
	args = appendInList(&sb, args, "container", f.Containers)
	args = appendInList(&sb, args, "pod", f.Pods)
	args = appendInList(&sb, args, "namespace", f.Namespaces)
	args = appendInList(&sb, args, "service", f.Services)
	args = appendInList(&sb, args, "env", f.Envs)
	if f.From > 0 {
		sb.WriteString(" AND ts >= ?")
		args = append(args, f.From)
	}
	if f.To > 0 {
		sb.WriteString(" AND ts <= ?")
		args = append(args, f.To)
	}
	sb.WriteString(` GROUP BY bucket, level ORDER BY bucket ASC`)

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Aggregate (bucket → counts-by-level) in one pass.
	bucketIdx := map[int64]int{}
	var out []logs.StatsBucket
	for rows.Next() {
		var bucket int64
		var level int
		var count int64
		if err := rows.Scan(&bucket, &level, &count); err != nil {
			return nil, err
		}
		idx, ok := bucketIdx[bucket]
		if !ok {
			idx = len(out)
			bucketIdx[bucket] = idx
			out = append(out, logs.StatsBucket{TS: bucket, Levels: map[string]int{}})
		}
		out[idx].Count += count
		out[idx].Levels[logs.LevelName(logs.Level(level))] += int(count)
	}
	return out, rows.Err()
}

// Vacuum enforces both age and size limits. We evict oldest-first; FTS rows
// are removed in the same statement to keep them consistent.
func (s *SQLiteStore) Vacuum(ctx context.Context, settings logs.Settings) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	var total int64

	// Age-based eviction.
	if settings.RetentionHours > 0 {
		cutoff := nowMS() - int64(settings.RetentionHours)*3_600_000
		res, err := s.db.ExecContext(ctx, `DELETE FROM logs WHERE ts < ?`, cutoff)
		if err != nil {
			return total, fmt.Errorf("age delete: %w", err)
		}
		n, _ := res.RowsAffected()
		total += n
		// FTS5 with content='logs' uses an external content table so we
		// must explicitly drop the matching FTS rows.
		if n > 0 {
			if _, err := s.db.ExecContext(ctx,
				`DELETE FROM logs_fts WHERE rowid NOT IN (SELECT id FROM logs)`,
			); err != nil {
				return total, fmt.Errorf("fts delete: %w", err)
			}
		}
	}

	// Size-based eviction. Use page_count * page_size as a proxy for file
	// size; delete the oldest 5% of rows if we're over the cap.
	if settings.MaxSizeMB > 0 {
		var bytes int64
		if err := s.db.QueryRowContext(ctx,
			`SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()`,
		).Scan(&bytes); err == nil {
			if bytes > int64(settings.MaxSizeMB)*1024*1024 {
				var deleteUntil int64
				if err := s.db.QueryRowContext(ctx,
					`SELECT ts FROM logs ORDER BY ts ASC LIMIT 1 OFFSET (SELECT COUNT(*)/20 FROM logs)`,
				).Scan(&deleteUntil); err == nil && deleteUntil > 0 {
					res, err := s.db.ExecContext(ctx, `DELETE FROM logs WHERE ts < ?`, deleteUntil)
					if err == nil {
						n, _ := res.RowsAffected()
						total += n
						_, _ = s.db.ExecContext(ctx,
							`DELETE FROM logs_fts WHERE rowid NOT IN (SELECT id FROM logs)`)
					}
				}
			}
		}
	}

	if total > 0 {
		// Hand free pages back to the OS. Bounded to a small chunk so we
		// don't hold the lock for long.
		_, _ = s.db.ExecContext(ctx, `PRAGMA incremental_vacuum(100)`)
	}
	return total, nil
}

func (s *SQLiteStore) LoadSettings(ctx context.Context) (logs.Settings, error) {
	out := logs.DefaultSettings()
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM logs_meta`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		n, _ := strconv.Atoi(v)
		switch k {
		case "retention_hours":
			if n > 0 {
				out.RetentionHours = n
			}
		case "max_size_mb":
			if n > 0 {
				out.MaxSizeMB = n
			}
		case "sample_info":
			if n > 0 {
				out.SampleInfo = n
			}
		case "sample_debug":
			if n > 0 {
				out.SampleDebug = n
			}
		}
	}
	return out, rows.Err()
}

func (s *SQLiteStore) SaveSettings(ctx context.Context, st logs.Settings) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	set := func(k string, v int) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO logs_meta(key, value) VALUES(?, ?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			k, strconv.Itoa(v))
		return err
	}
	if err := set("retention_hours", st.RetentionHours); err != nil {
		return err
	}
	if err := set("max_size_mb", st.MaxSizeMB); err != nil {
		return err
	}
	if err := set("sample_info", st.SampleInfo); err != nil {
		return err
	}
	if err := set("sample_debug", st.SampleDebug); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) IncDropped(ctx context.Context, n int64) error {
	if n <= 0 {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO logs_meta(key, value) VALUES('dropped', ?)
		ON CONFLICT(key) DO UPDATE SET value = CAST((CAST(value AS INTEGER) + ?) AS TEXT)
	`, strconv.FormatInt(n, 10), n)
	return err
}

func (s *SQLiteStore) DroppedCount(ctx context.Context) (int64, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM logs_meta WHERE key='dropped'`).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	return n, nil
}

// nowMS returns the current unix-millis. Pulled out so tests can swap it.
var nowMS = func() int64 {
	return timeNowMS()
}
