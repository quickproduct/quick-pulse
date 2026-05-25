package workers

import (
	"database/sql"
	"log"
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

	"quickpulse/backend/db"
	"quickpulse/backend/models"
	"quickpulse/backend/ws"
)

func StartMetricsWorker() {
	intervalSecs := 10
	if envSecs := os.Getenv("METRICS_COLLECTION_INTERVAL_SECONDS"); envSecs != "" {
		if val, err := strconv.Atoi(envSecs); err == nil {
			intervalSecs = val
		}
	}

	log.Printf("Starting metrics worker with interval %d seconds", intervalSecs)
	go func() {
		consecutiveErrors := 0
		for {
			err := collectAndProcessMetrics()
			if err != nil {
				consecutiveErrors++
				log.Printf("Metrics worker error (consecutive %d): %v", consecutiveErrors, err)
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
		log.Printf("Host registered: %s (IP: %s)", hostname, ipAddress)
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
		log.Printf("Warning: failed to store host metric: %v", err)
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
		log.Printf("Failed to query alert rules: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, metricType, operator string
		var threshold float64
		var durationSeconds int
		if err := rows.Scan(&id, &metricType, &threshold, &operator, &durationSeconds); err != nil {
			log.Printf("Failed to scan alert rule: %v", err)
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
				log.Printf("Failed to create alert: %v", err)
			} else {
				log.Printf("Alert triggered! Severity: %s, Message: %s", severity, message)
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
