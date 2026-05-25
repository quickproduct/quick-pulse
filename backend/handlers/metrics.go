package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"quickpulse/backend/db"
	"quickpulse/backend/models"
)

// GetMetricsSummaryHandler handles GET /api/v1/metrics/summary
func GetMetricsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	// Query current host
	var hostID string
	err := db.DB.QueryRow("SELECT id FROM hosts LIMIT 1").Scan(&hostID)
	if err == sql.ErrNoRows {
		WriteJSON(w, http.StatusOK, map[string]interface{}{})
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error querying host")
		return
	}

	// Query latest metric
	var m models.MetricSnapshot
	var timeStr string
	err = db.DB.QueryRow(`
		SELECT time, cpu_percent, memory_percent, memory_used, memory_total, disk_percent,
		       net_bytes_sent, net_bytes_recv, load_1m, load_5m, load_15m, process_count, uptime_seconds
		FROM host_metrics
		WHERE host_id = ?
		ORDER BY time DESC
		LIMIT 1
	`, hostID).Scan(
		&timeStr, &m.CPUPercent, &m.MemoryPercent, &m.MemoryUsed, &m.MemoryTotal, &m.DiskPercent,
		&m.NetBytesSent, &m.NetBytesRecv, &m.Load1m, &m.Load5m, &m.Load15m, &m.ProcessCount, &m.UptimeSeconds,
	)

	if err == sql.ErrNoRows {
		WriteJSON(w, http.StatusOK, map[string]interface{}{})
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error querying metrics")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"cpu_percent":    m.CPUPercent,
		"memory_percent": m.MemoryPercent,
		"memory_used":    m.MemoryUsed,
		"memory_total":   m.MemoryTotal,
		"disk_percent":   m.DiskPercent,
		"net_bytes_sent": m.NetBytesSent,
		"net_bytes_recv": m.NetBytesRecv,
		"load_1m":        m.Load1m,
		"load_5m":        m.Load5m,
		"load_15m":       m.Load15m,
		"process_count":  m.ProcessCount,
		"uptime_seconds": m.UptimeSeconds,
	})
}

// GetMetricsHistoryHandler handles GET /api/v1/metrics/history
func GetMetricsHistoryHandler(w http.ResponseWriter, r *http.Request) {
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "cpu"
	}
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "1h"
	}

	validMetrics := map[string]string{
		"cpu":     "cpu_percent",
		"memory":  "memory_percent",
		"disk":    "disk_percent",
		"network": "net_bytes_recv",
		"load":    "load_1m",
	}

	col, ok := validMetrics[metric]
	if !ok {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid metric '%s'. Valid: cpu, memory, disk, network, load", metric))
		return
	}

	var delta time.Duration
	var bucketExpr string

	switch rangeKey {
	case "15m":
		delta = 15 * time.Minute
		bucketExpr = "strftime('%Y-%m-%dT%H:%M:00Z', time)"
	case "1h":
		delta = time.Hour
		bucketExpr = "strftime('%Y-%m-%dT%H:%M:00Z', time)"
	case "6h":
		delta = 6 * time.Hour
		bucketExpr = "strftime('%Y-%m-%dT%H:', time) || printf('%02d:00Z', (cast(strftime('%M', time) as integer) / 5) * 5)"
	case "24h":
		delta = 24 * time.Hour
		bucketExpr = "strftime('%Y-%m-%dT%H:', time) || printf('%02d:00Z', (cast(strftime('%M', time) as integer) / 15) * 15)"
	case "7d":
		delta = 7 * 24 * time.Hour
		bucketExpr = "strftime('%Y-%m-%dT', time) || printf('%02d:00:00Z', (cast(strftime('%H', time) as integer) / 2) * 2)"
	default:
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid range '%s'. Valid: 15m, 1h, 6h, 24h, 7d", rangeKey))
		return
	}

	// Find the current host ID
	var hostID string
	err := db.DB.QueryRow("SELECT id FROM hosts LIMIT 1").Scan(&hostID)
	if err == sql.ErrNoRows {
		WriteJSON(w, http.StatusOK, models.MetricHistoryResponse{
			Metric: metric,
			Range:  rangeKey,
			Data:   []models.MetricHistoryPoint{},
		})
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error querying host")
		return
	}

	endTime := time.Now().UTC()
	startTime := endTime.Add(-delta)

	startTimeStr := startTime.Format("2006-01-02 15:04:05")
	endTimeStr := endTime.Format("2006-01-02 15:04:05")

	// Build query
	query := fmt.Sprintf(`
		SELECT %s AS bucket, AVG(%s) AS val
		FROM host_metrics
		WHERE host_id = ? AND time >= ? AND time <= ?
		GROUP BY bucket
		ORDER BY bucket
	`, bucketExpr, col)

	rows, err := db.DB.Query(query, hostID, startTimeStr, endTimeStr)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query history: %v", err))
		return
	}
	defer rows.Close()

	points := []models.MetricHistoryPoint{}
	for rows.Next() {
		var bucket string
		var val float64
		if err := rows.Scan(&bucket, &val); err != nil {
			WriteError(w, http.StatusInternalServerError, "Error scanning metrics history")
			return
		}
		// Format to 2 decimal places
		val = float64(int(val*100)) / 100
		points = append(points, models.MetricHistoryPoint{
			Time:  bucket,
			Value: val,
		})
	}

	WriteJSON(w, http.StatusOK, models.MetricHistoryResponse{
		Metric: metric,
		Range:  rangeKey,
		Data:   points,
	})
}
