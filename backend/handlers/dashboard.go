package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"

	"quickpulse/backend/db"
	"quickpulse/backend/models"
)

// GetDashboardHandler handles GET /api/v1/dashboard
func GetDashboardHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch host info
	var h models.Host
	var hostID string
	err := db.DB.QueryRow(`
		SELECT id, hostname, ip_address, os_info, cpu_count, total_memory, total_disk
		FROM hosts
		LIMIT 1
	`).Scan(&hostID, &h.Hostname, &h.IPAddress, &h.OSInfo, &h.CPUCount, &h.TotalMemory, &h.TotalDisk)

	var hostPtr *models.Host
	if err == nil {
		hostPtr = &h
	}

	// 2. Fetch latest metrics
	var m models.MetricSnapshot
	var metricsPtr *models.MetricSnapshot
	if hostPtr != nil {
		var timeStr string
		err = db.DB.QueryRow(`
			SELECT cpu_percent, memory_percent, memory_used, memory_total, disk_percent,
			       net_bytes_sent, net_bytes_recv, load_1m, load_5m, load_15m, process_count, uptime_seconds, time
			FROM host_metrics
			WHERE host_id = ?
			ORDER BY time DESC
			LIMIT 1
		`, hostID).Scan(
			&m.CPUPercent, &m.MemoryPercent, &m.MemoryUsed, &m.MemoryTotal, &m.DiskPercent,
			&m.NetBytesSent, &m.NetBytesRecv, &m.Load1m, &m.Load5m, &m.Load15m, &m.ProcessCount, &m.UptimeSeconds, &timeStr,
		)
		if err == nil {
			t, _ := time.Parse("2006-01-02 15:04:05", timeStr)
			m.Timestamp = t
			metricsPtr = &m
		}
	}

	// 3. Fetch container summary from Docker
	containerSummary := map[string]int{"total": 0, "running": 0, "stopped": 0}
	stacksMap := make(map[string]map[string]string) // name -> service_name -> state

	cli, err := getDockerClient()
	if err == nil {
		defer cli.Close()
		containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
		if err == nil {
			for _, c := range containers {
				cName := ""
				if len(c.Names) > 0 {
					cName = strings.TrimPrefix(c.Names[0], "/")
				} else {
					cName = c.ID[:12]
				}

				project := c.Labels["com.docker.compose.project"]

				if strings.HasPrefix(cName, "qp-") || project == "quickpulse" {
					continue
				}

				containerSummary["total"]++
				if c.State == "running" {
					containerSummary["running"]++
				} else {
					containerSummary["stopped"]++
				}

				if project != "" {
					if _, exists := stacksMap[project]; !exists {
						stacksMap[project] = make(map[string]string)
					}
					serviceName := c.Labels["com.docker.compose.service"]
					if serviceName == "" {
						serviceName = cName
					}
					stacksMap[project][serviceName] = c.State
				}
			}
		}
	}

	// 4. Calculate stack summary
	stackSummary := map[string]int{"total": len(stacksMap), "running": 0, "partial": 0, "stopped": 0}
	for _, services := range stacksMap {
		running := 0
		total := len(services)
		for _, state := range services {
			if state == "running" {
				running++
			}
		}

		if running == total && total > 0 {
			stackSummary["running"]++
		} else if running > 0 {
			stackSummary["partial"]++
		} else {
			stackSummary["stopped"]++
		}
	}

	// 5. Fetch recent events from SQLite
	recentEvents := []models.EventResponse{}
	rows, err := db.DB.Query(`
		SELECT id, container_docker_id, container_name, event_type, timestamp, metadata
		FROM container_events
		ORDER BY timestamp DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e models.EventResponse
			var cID, cName, metadataStr sql.NullString
			var timeStr string
			err := rows.Scan(&e.ID, &cID, &cName, &e.EventType, &timeStr, &metadataStr)
			if err == nil {
				if cID.Valid {
					e.ContainerDockerID = cID.String
				}
				if cName.Valid {
					e.ContainerName = cName.String
				}
				t, _ := time.Parse("2006-01-02 15:04:05", timeStr)
				e.Timestamp = &t
				if metadataStr.Valid && metadataStr.String != "" {
					_ = json.Unmarshal([]byte(metadataStr.String), &e.Metadata)
				}
				recentEvents = append(recentEvents, e)
			}
		}
	}

	resp := models.DashboardResponse{
		Host:         hostPtr,
		Metrics:      metricsPtr,
		Containers:   containerSummary,
		RecentEvents: recentEvents,
		StackSummary: stackSummary,
	}

	WriteJSON(w, http.StatusOK, resp)
}
