package handlers

import (
	"github.com/go-chi/chi/v5"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quickpulse/backend/internal/logs"
	"quickpulse/backend/internal/logs/pubsub"
	logsservice "quickpulse/backend/internal/logs/service"
)

// LogsService is set from main.go after service.Start() returns. Keeping it
// as a package-level pointer avoids threading a service struct through
// every handler signature — same pattern as `db.DB`.
var LogsService *logsservice.Service

// SearchLogsHandler handles GET /api/v1/logs — paginated newest-first
// search with optional FTS, filters, and cursor.
func SearchLogsHandler(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		WriteError(w, http.StatusServiceUnavailable, "Logs service not initialized")
		return
	}
	f := parseFilter(r)
	entries, cursor, err := LogsService.Store.Query(r.Context(), f)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Query failed: "+err.Error())
		return
	}
	out := make([]logs.PublicEntry, len(entries))
	for i, e := range entries {
		out[i] = e.ToPublic()
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"logs":        out,
		"next_cursor": cursor,
	})
}

// GetLogHandler handles GET /api/v1/logs/{id} — single-entry detail.
func GetLogHandler(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		WriteError(w, http.StatusServiceUnavailable, "Logs service not initialized")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	e, err := LogsService.Store.Get(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Log entry not found")
		return
	}
	WriteJSON(w, http.StatusOK, e.ToPublic())
}

// SourcesHandler handles GET /api/v1/logs/sources — distinct facet values
// for the filter dropdowns.
func SourcesHandler(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		WriteError(w, http.StatusServiceUnavailable, "Logs service not initialized")
		return
	}
	resp, err := LogsService.Store.Sources(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Sources failed: "+err.Error())
		return
	}
	// Also surface active stream counts so the UI can show coverage.
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"containers":     resp.Containers,
		"pods":           resp.Pods,
		"namespaces":     resp.Namespaces,
		"services":       resp.Services,
		"envs":           resp.Envs,
		"clusters":       resp.Clusters,
		"platforms":      resp.Platforms,
		"dropped":        resp.Dropped,
		"active_streams": LogsService.Reg.Size(),
	})
}

// StatsHandler handles GET /api/v1/logs/stats — time-bucketed counts for
// the sparkline / histogram.
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		WriteError(w, http.StatusServiceUnavailable, "Logs service not initialized")
		return
	}
	f := parseFilter(r)
	bucket := r.URL.Query().Get("bucket")
	var bucketMs int64 = 60_000
	switch bucket {
	case "5m":
		bucketMs = 300_000
	case "1h":
		bucketMs = 3_600_000
	case "10s":
		bucketMs = 10_000
	}
	out, err := LogsService.Store.Stats(r.Context(), f, bucketMs)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Stats failed: "+err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, out)
}

// SettingsGetHandler / SettingsPutHandler — admin-only knob for retention
// and sampling. Reuses AdminMiddleware.
func SettingsGetHandler(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		WriteError(w, http.StatusServiceUnavailable, "Logs service not initialized")
		return
	}
	s, err := LogsService.Store.LoadSettings(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Settings load failed: "+err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, s)
}

func SettingsPutHandler(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		WriteError(w, http.StatusServiceUnavailable, "Logs service not initialized")
		return
	}
	var s logs.Settings
	if err := ParseJSON(r, &s); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	// Sanity guards — the UI also validates but a quick re-check here
	// stops mistakes from disabling retention entirely.
	if s.RetentionHours < 1 {
		s.RetentionHours = 1
	}
	if s.MaxSizeMB < 10 {
		s.MaxSizeMB = 10
	}
	if s.SampleInfo < 1 {
		s.SampleInfo = 1
	}
	if s.SampleDebug < 1 {
		s.SampleDebug = 1
	}
	if err := LogsService.ApplySettings(r.Context(), s); err != nil {
		WriteError(w, http.StatusInternalServerError, "Save failed: "+err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, s)
}

// parseFilter pulls a logs.Filter out of the query string. We accept both
// comma-separated values and repeated params so the UI can use whichever
// is more convenient.
func parseFilter(r *http.Request) logs.Filter {
	q := r.URL.Query()
	f := logs.Filter{
		Query:  q.Get("q"),
		Cursor: q.Get("cursor"),
	}

	for _, v := range splitCSV(q["level"]) {
		f.Levels = append(f.Levels, logs.ParseLevel(v))
	}
	for _, v := range splitCSV(q["platform"]) {
		if p, ok := logs.ParsePlatform(v); ok {
			f.Platforms = append(f.Platforms, p)
		}
	}
	f.Clusters = splitCSV(q["cluster"])
	f.Containers = splitCSV(q["container"])
	f.Pods = splitCSV(q["pod"])
	f.Namespaces = splitCSV(q["namespace"])
	f.Services = splitCSV(q["service"])
	f.Envs = splitCSV(q["env"])

	if v := q.Get("from"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.From = n
		}
	}
	if v := q.Get("to"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.To = n
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	return f
}

// splitCSV flattens both `?level=INFO&level=WARN` and `?level=INFO,WARN`
// into a single slice. Trims whitespace; drops empties.
func splitCSV(vals []string) []string {
	var out []string
	for _, raw := range vals {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// HandleWSLogsStream handles GET /ws/logs/stream — live tail with filter
// updates and pause/resume.
//
// Client wire protocol (JSON):
//
//	{ "action": "subscribe", "filter": { level: ["ERROR"], ... } }
//	{ "action": "filter",    "filter": { ... } }   // change filter live
//	{ "action": "pause" }
//	{ "action": "resume" }
//
// Server messages:
//
//	{ "logs":    [PublicEntry, ...] }     // a batch
//	{ "dropped": <int> }                  // backpressure signal
//	{ "error":   "..." }                  // fatal; server is about to close
func HandleWSLogsStream(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("Logs service not initialized"))
		return
	}
	if _, ok := validateWSAuth(r); !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.L().Sugar().Infof("WS upgrade failed for logs stream: %v", err)
		return
	}
	defer conn.Close()

	// Initial subscribe with empty filter — clients always send a follow-
	// up with their actual filter immediately after open.
	sub := LogsService.Broker.Subscribe(pubsub.SubscriberFilter{}, 64)
	if sub == nil {
		_ = conn.WriteJSON(map[string]string{"error": "max subscribers reached"})
		return
	}
	defer LogsService.Broker.Unsubscribe(sub)

	// Writer goroutine: drain the subscriber's Out channel onto the WS.
	doneWrite := make(chan struct{})
	go func() {
		defer close(doneWrite)
		// Surface dropped-counter to client periodically when non-zero.
		dropTick := time.NewTicker(2 * time.Second)
		defer dropTick.Stop()
		var lastReportedDrop int64

		for {
			select {
			case batch, ok := <-sub.Out:
				if !ok {
					return
				}
				if err := conn.WriteJSON(map[string]interface{}{"logs": batch}); err != nil {
					return
				}
			case <-dropTick.C:
				cur := sub.Dropped
				if cur > lastReportedDrop {
					delta := cur - lastReportedDrop
					lastReportedDrop = cur
					if err := conn.WriteJSON(map[string]int64{"dropped": delta}); err != nil {
						return
					}
				}
			}
		}
	}()

	// Reader loop: parse client commands and apply to the subscriber.
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var msg struct {
			Action string                 `json:"action"`
			Filter map[string]interface{} `json:"filter"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Action {
		case "subscribe", "filter":
			sub.SetFilter(filterFromMap(msg.Filter))
		case "pause":
			sub.Pause()
		case "resume":
			sub.Resume()
		}
	}
	<-doneWrite
}

// filterFromMap parses the JSON-shaped filter sent by the client into the
// in-memory SubscriberFilter used by the broker. Permissive — unknown keys
// are silently ignored.
func filterFromMap(m map[string]interface{}) pubsub.SubscriberFilter {
	var f pubsub.SubscriberFilter
	for _, v := range strSlice(m["level"]) {
		f.Levels = append(f.Levels, logs.ParseLevel(v))
	}
	for _, v := range strSlice(m["platform"]) {
		if p, ok := logs.ParsePlatform(v); ok {
			f.Platforms = append(f.Platforms, p)
		}
	}
	f.Clusters = strSlice(m["cluster"])
	f.Containers = strSlice(m["container"])
	f.Pods = strSlice(m["pod"])
	f.Namespaces = strSlice(m["namespace"])
	f.Services = strSlice(m["service"])
	f.Envs = strSlice(m["env"])
	if q, ok := m["q"].(string); ok {
		f.Query = q
	}
	return f
}

func strSlice(v interface{}) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		return splitCSV([]string{t})
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// ExportLogsHandler handles GET /api/v1/logs/export — streams CSV or JSON
// of the current filter without buffering the whole result set. Useful for
// quick offline analysis (Sentry- / Better Stack-style "download").
func ExportLogsHandler(w http.ResponseWriter, r *http.Request) {
	if LogsService == nil {
		WriteError(w, http.StatusServiceUnavailable, "Logs service not initialized")
		return
	}
	f := parseFilter(r)
	if f.Limit <= 0 {
		f.Limit = 500 // chunk size; we paginate through the cursor
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	filename := fmt.Sprintf("logs-%d.%s", time.Now().Unix(), format)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("ts,level,platform,source_id,container,pod,namespace,service,host,env,trace_id,message\n"))
		streamPages(r, f, func(e logs.Entry) {
			_, _ = fmt.Fprintf(w, "%d,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%q\n",
				e.TS, logs.LevelName(e.Level), logs.PlatformName(e.Platform),
				e.SourceID, csvSafe(e.Container), csvSafe(e.Pod), csvSafe(e.Namespace),
				csvSafe(e.Service), csvSafe(e.Host), csvSafe(e.Env), csvSafe(e.TraceID),
				e.Message,
			)
		})
	default:
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		_, _ = w.Write([]byte("[\n"))
		first := true
		streamPages(r, f, func(e logs.Entry) {
			if !first {
				_, _ = w.Write([]byte(",\n"))
			}
			first = false
			_ = enc.Encode(e.ToPublic())
		})
		_, _ = w.Write([]byte("]\n"))
	}
}

// streamPages walks the query in chunks of f.Limit until cursor is empty
// or we hit a hard cap (50k rows) — protects against accidentally exporting
// the whole table on a busy host.
func streamPages(r *http.Request, f logs.Filter, emit func(logs.Entry)) {
	const cap = 50_000
	seen := 0
	for {
		entries, next, err := LogsService.Store.Query(r.Context(), f)
		if err != nil {
			return
		}
		for _, e := range entries {
			emit(e)
			seen++
			if seen >= cap {
				return
			}
		}
		if next == "" {
			return
		}
		f.Cursor = next
	}
}

func csvSafe(s string) string {
	// CSV: replace commas/newlines/quotes to keep the simple writer happy.
	r := strings.NewReplacer(",", " ", "\n", " ", "\"", "'")
	return r.Replace(s)
}
