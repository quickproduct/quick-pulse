package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"quickpulse/backend/db"
	"quickpulse/backend/models"
)

// GetEventsHandler handles GET /api/v1/events
func GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := db.DB.Query(`
		SELECT id, container_docker_id, container_name, event_type, timestamp, metadata
		FROM container_events
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to load events")
		return
	}
	defer rows.Close()

	events := []models.EventResponse{}
	for rows.Next() {
		var e models.EventResponse
		var cID, cName, metadataStr sql.NullString
		var timeStr string
		err := rows.Scan(&e.ID, &cID, &cName, &e.EventType, &timeStr, &metadataStr)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Error reading events")
			return
		}
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
		} else {
			e.Metadata = make(map[string]interface{})
		}
		events = append(events, e)
	}

	WriteJSON(w, http.StatusOK, events)
}
