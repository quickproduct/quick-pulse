package workers

import (
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	netutil "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"quickpulse/backend/internal/db"
	"quickpulse/backend/internal/models"
	"quickpulse/backend/internal/ws"
)

func StartMetricsWorker() {
	intervalSecs := 10
	if envSecs := os.Getenv("METRICS_COLLECTION_INTERVAL_SECONDS"); envSecs != "" {
		if val, err := strconv.Atoi(envSecs); err == nil {
			intervalSecs = val
		}
	}

	zap.L().Sugar().Infof("Starting metrics worker with interval %d seconds", intervalSecs)
	go func() {
		consecutiveErrors := 0
		for {
			err := collectAndProcessMetrics()
			if err != nil {
				consecutiveErrors++
				zap.L().Sugar().Infof("Metrics worker error (consecutive %d): %v", consecutiveErrors, err)
				backoff := 5 * consecutiveErrors
				if backoff > 300 {
					backoff = 300
				}
				time.Sleep(time.Duration(backoff) * time.Second)
				continue
			}
			consecutiveErrors = 0
			time.Sleep(time.Duration(intervalSecs) * time.Second)
		}
	}()
}

func collectAndProcessMetrics() error {
	// 1. Get host details and upsert
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	ipAddress := "127.0.0.1"
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ipAddress = ipnet.IP.String()
					break
				}
			}
		}
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	dUsage, err := disk.Usage("/")
	if err != nil {
		return err
	}

	var hostID string
	err = db.DB.QueryRow("SELECT id FROM hosts WHERE hostname = ?", hostname).Scan(&hostID)
	if err == sql.ErrNoRows {
		hostID = uuid.New().String()
		_, err = db.DB.Exec(
			"INSERT INTO hosts (id, hostname, ip_address, os_info, cpu_count, total_memory, total_disk) VALUES (?, ?, ?, ?, ?, ?, ?)",
			hostID, hostname, ipAddress, runtime.GOOS, runtime.NumCPU(), vm.Total, dUsage.Total,
		)
		if err != nil {
			return err
		}
		zap.L().Sugar().Infof("Host registered: %s (IP: %s)", hostname, ipAddress)
	} else if err != nil {
		return err
	}

	// 2. Collect host metrics
	cpuPercents, err := cpu.Percent(500*time.Millisecond, false)
	var cpuPercent float64
	if err == nil && len(cpuPercents) > 0 {
		cpuPercent = cpuPercents[0]
	}

	diskIO, _ := disk.IOCounters()
	var diskReadBytes, diskWriteBytes uint64
	for _, io := range diskIO {
		diskReadBytes += io.ReadBytes
		diskWriteBytes += io.WriteBytes
	}

	netIO, _ := netutil.IOCounters(false)
	var netBytesSent, netBytesRecv uint64
	if len(netIO) > 0 {
		netBytesSent = netIO[0].BytesSent
		netBytesRecv = netIO[0].BytesRecv
	}

	loadAvg, err := load.Avg()
	if err != nil {
		loadAvg = &load.AvgStat{}
	}

	pids, err := process.Pids()
	processCount := len(pids)

	hUptime, err := host.Uptime()
	if err != nil {
		hUptime = 0
	}

	snapshot := models.MetricSnapshot{
		CPUPercent:     cpuPercent,
		MemoryPercent:  vm.UsedPercent,
		MemoryUsed:     vm.Used,
		MemoryTotal:    vm.Total,
		DiskPercent:    dUsage.UsedPercent,
		DiskReadBytes:  diskReadBytes,
		DiskWriteBytes: diskWriteBytes,
		NetBytesSent:   netBytesSent,
		NetBytesRecv:   netBytesRecv,
		Load1m:         loadAvg.Load1,
		Load5m:         loadAvg.Load5,
		Load15m:        loadAvg.Load15,
		ProcessCount:   processCount,
		UptimeSeconds:  int(hUptime),
		Timestamp:      time.Now().UTC(),
	}

	// 3. Store metric in SQLite
	_, err = db.DB.Exec(
		`INSERT INTO host_metrics (
			time, host_id, cpu_percent, memory_percent, memory_used, memory_total,
			disk_percent, disk_read_bytes, disk_write_bytes, net_bytes_sent, net_bytes_recv,
			load_1m, load_5m, load_15m, process_count, uptime_seconds
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.Timestamp.Format("2006-01-02 15:04:05"), hostID, snapshot.CPUPercent, snapshot.MemoryPercent,
		snapshot.MemoryUsed, snapshot.MemoryTotal, snapshot.DiskPercent, snapshot.DiskReadBytes,
		snapshot.DiskWriteBytes, snapshot.NetBytesSent, snapshot.NetBytesRecv, snapshot.Load1m,
		snapshot.Load5m, snapshot.Load15m, snapshot.ProcessCount, snapshot.UptimeSeconds,
	)
	if err != nil {
		zap.L().Sugar().Infof("Warning: failed to store host metric: %v", err)
	}

	// 4. Evaluate alert rules
	evaluateAlertRules(snapshot)

	// 5. Broadcast to WebSocket clients
	broadcastData := map[string]interface{}{
		"cpu_percent":      snapshot.CPUPercent,
		"memory_percent":   snapshot.MemoryPercent,
		"memory_used":      snapshot.MemoryUsed,
		"memory_total":     snapshot.MemoryTotal,
		"disk_percent":     snapshot.DiskPercent,
		"disk_read_bytes":  snapshot.DiskReadBytes,
		"disk_write_bytes": snapshot.DiskWriteBytes,
		"net_bytes_sent":   snapshot.NetBytesSent,
		"net_bytes_recv":   snapshot.NetBytesRecv,
		"load_1m":          snapshot.Load1m,
		"load_5m":          snapshot.Load5m,
		"load_15m":         snapshot.Load15m,
		"process_count":    snapshot.ProcessCount,
		"uptime_seconds":   snapshot.UptimeSeconds,
		"timestamp":        snapshot.Timestamp.Format(time.RFC3339),
	}
	ws.Manager.Broadcast("metrics", broadcastData)

	return nil
}

func evaluateAlertRules(metrics models.MetricSnapshot) {
	rows, err := db.DB.Query("SELECT id, metric_type, threshold, operator, duration_seconds FROM alert_rules WHERE enabled = 1")
	if err != nil {
		zap.L().Sugar().Infof("Failed to query alert rules: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, metricType, operator string
		var threshold float64
		var durationSeconds int
		if err := rows.Scan(&id, &metricType, &threshold, &operator, &durationSeconds); err != nil {
			zap.L().Sugar().Infof("Failed to scan alert rule: %v", err)
			continue
		}

		var value float64
		switch metricType {
		case "cpu":
			value = metrics.CPUPercent
		case "memory":
			value = metrics.MemoryPercent
		case "disk":
			value = metrics.DiskPercent
		case "load":
			value = metrics.Load1m
		default:
			continue
		}

		triggered := false
		switch operator {
		case "gt":
			triggered = value > threshold
		case "gte":
			triggered = value >= threshold
		case "lt":
			triggered = value < threshold
		case "lte":
			triggered = value <= threshold
		case "eq":
			triggered = value == threshold
		}

		if triggered {
			severity := "warning"
			if threshold >= 90 {
				severity = "critical"
			}
			message := metricType + " is " + strconv.FormatFloat(value, 'f', 2, 64) + "% (threshold: " + strconv.FormatFloat(threshold, 'f', 2, 64) + "% " + operator + ")"
			alertID := uuid.New().String()

			_, err := db.DB.Exec(
				"INSERT INTO alerts (id, rule_id, severity, message, acknowledged) VALUES (?, ?, ?, ?, 0)",
				alertID, id, severity, message,
			)
			if err != nil {
				zap.L().Sugar().Infof("Failed to create alert: %v", err)
			} else {
				zap.L().Sugar().Infof("Alert triggered! Severity: %s, Message: %s", severity, message)
				// Broadcast alert event to websocket clients
				ws.Manager.Broadcast("events", map[string]interface{}{
					"id":         alertID,
					"event_type": "alert_triggered",
					"timestamp":  time.Now().UTC().Format(time.RFC3339),
					"metadata": map[string]interface{}{
						"rule_id":  id,
						"severity": severity,
						"message":  message,
					},
				})
			}
		}
	}
}

// StartMetricsJanitorWorker starts a background loop to prune and downsample metrics.
func StartMetricsJanitorWorker() {
	zap.L().Sugar().Info("Starting metrics janitor worker...")
	go func() {
		// Run on startup, then every 12 hours
		for {
			zap.L().Sugar().Info("Metrics janitor: running downsampling and cleanup...")
			if err := runMetricsJanitor(); err != nil {
				zap.L().Sugar().Infof("Metrics janitor error: %v", err)
			}
			time.Sleep(12 * time.Hour)
		}
	}()
}

func runMetricsJanitor() error {
	retentionDays := 30
	if envVal := os.Getenv("METRICS_RETENTION_DAYS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			retentionDays = val
		}
	}

	downsampleHours := 24
	if envVal := os.Getenv("METRICS_DOWNSAMPLE_THRESHOLD_HOURS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			downsampleHours = val
		}
	}

	// 1. Delete metrics older than retentionDays
	pruneQuery := fmt.Sprintf("DELETE FROM host_metrics WHERE time < datetime('now', '-%d days')", retentionDays)
	_, err := db.DB.Exec(pruneQuery)
	if err != nil {
		return fmt.Errorf("failed to prune old metrics: %w", err)
	}

	// 2. Downsample metrics older than downsampleHours (group by hour and host_id)
	downsampleQuery := fmt.Sprintf(`
		SELECT strftime('%%Y-%%m-%%d %%H:00:00', time) as hour_bucket, host_id
		FROM host_metrics
		WHERE time < datetime('now', '-%d hours')
		GROUP BY hour_bucket, host_id
		HAVING count(*) > 1
	`, downsampleHours)
	rows, err := db.DB.Query(downsampleQuery)
	if err != nil {
		return fmt.Errorf("failed to query raw metrics for downsampling: %w", err)
	}
	defer rows.Close()

	type downsampleTarget struct {
		hourBucket string
		hostID     string
	}
	var targets []downsampleTarget
	for rows.Next() {
		var t downsampleTarget
		if err := rows.Scan(&t.hourBucket, &t.hostID); err != nil {
			zap.L().Sugar().Infof("Metrics janitor: error scanning row: %v", err)
			continue
		}
		targets = append(targets, t)
	}

	zap.L().Sugar().Infof("Metrics janitor: found %d hourly buckets to downsample", len(targets))

	// Downsample each bucket sequentially in a transaction to minimize database locking
	for _, target := range targets {
		// Introduce a tiny pause to yield CPU and database locks to regular requests
		time.Sleep(50 * time.Millisecond)

		tx, err := db.DB.Begin()
		if err != nil {
			zap.L().Sugar().Infof("Metrics janitor: failed to begin transaction: %v", err)
			continue
		}

		// Calculate averages for this host and hour
		var avgCPU, avgMem, avgDisk, avgLoad1m, avgLoad5m, avgLoad15m float64
		var avgMemUsed, avgMemTotal, avgDiskRead, avgDiskWrite, avgNetSent, avgNetRecv, avgProcesses, avgUptime float64
		err = tx.QueryRow(`
			SELECT 
				AVG(cpu_percent), AVG(memory_percent), AVG(memory_used), AVG(memory_total),
				AVG(disk_percent), AVG(disk_read_bytes), AVG(disk_write_bytes),
				AVG(net_bytes_sent), AVG(net_bytes_recv),
				AVG(load_1m), AVG(load_5m), AVG(load_15m),
				AVG(process_count), AVG(uptime_seconds)
			FROM host_metrics
			WHERE host_id = ? AND time >= ? AND time < datetime(?, '+1 hour')
		`, target.hostID, target.hourBucket, target.hourBucket).Scan(
			&avgCPU, &avgMem, &avgMemUsed, &avgMemTotal,
			&avgDisk, &avgDiskRead, &avgDiskWrite,
			&avgNetSent, &avgNetRecv,
			&avgLoad1m, &avgLoad5m, &avgLoad15m,
			&avgProcesses, &avgUptime,
		)
		if err != nil {
			tx.Rollback()
			zap.L().Sugar().Infof("Metrics janitor: failed to calculate averages for %s (host %s): %v", target.hourBucket, target.hostID, err)
			continue
		}

		// Delete high-resolution rows for this hour and host
		_, err = tx.Exec(`
			DELETE FROM host_metrics
			WHERE host_id = ? AND time >= ? AND time < datetime(?, '+1 hour')
		`, target.hostID, target.hourBucket, target.hourBucket)
		if err != nil {
			tx.Rollback()
			zap.L().Sugar().Infof("Metrics janitor: failed to delete raw metrics for %s: %v", target.hourBucket, err)
			continue
		}

		// Insert a single downsampled hourly metric row
		_, err = tx.Exec(`
			INSERT INTO host_metrics (
				time, host_id, cpu_percent, memory_percent, memory_used, memory_total,
				disk_percent, disk_read_bytes, disk_write_bytes, net_bytes_sent, net_bytes_recv,
				load_1m, load_5m, load_15m, process_count, uptime_seconds
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, target.hourBucket, target.hostID, avgCPU, avgMem, int64(avgMemUsed), int64(avgMemTotal),
			avgDisk, int64(avgDiskRead), int64(avgDiskWrite), int64(avgNetSent), int64(avgNetRecv),
			avgLoad1m, avgLoad5m, avgLoad15m, int(avgProcesses), int(avgUptime))
		if err != nil {
			tx.Rollback()
			zap.L().Sugar().Infof("Metrics janitor: failed to insert downsampled metrics for %s: %v", target.hourBucket, err)
			continue
		}

		if err := tx.Commit(); err != nil {
			zap.L().Sugar().Infof("Metrics janitor: failed to commit transaction: %v", err)
		}
	}

	// 3. Run incremental vacuum to reclaim unused disk space safely
	if _, err := db.DB.Exec("PRAGMA incremental_vacuum(50)"); err != nil {
		zap.L().Sugar().Infof("Metrics janitor: warning: failed to run incremental vacuum: %v", err)
	}

	zap.L().Sugar().Info("Metrics janitor: cycle completed successfully")
	return nil
}
