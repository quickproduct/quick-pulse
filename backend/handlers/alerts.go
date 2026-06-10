package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"

	"quickpulse/backend/db"
	"quickpulse/backend/models"
)

// ListAlertsHandler handles GET /api/v1/alerts
func ListAlertsHandler(w http.ResponseWriter, r *http.Request) {
	// List active/recent alerts, ordering by created_at desc
	rows, err := db.DB.Query(`
		SELECT id, rule_id, severity, message, acknowledged, created_at
		FROM alerts
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to load alerts")
		return
	}
	defer rows.Close()

	alerts := []models.AlertResponse{}
	for rows.Next() {
		var a models.AlertResponse
		var ruleID sql.NullString
		var ackVal int
		var alertTime time.Time
		err := rows.Scan(&a.ID, &ruleID, &a.Severity, &a.Message, &ackVal, &alertTime)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Error reading alerts")
			return
		}
		if ruleID.Valid {
			a.RuleID = &ruleID.String
		}
		a.Acknowledged = ackVal != 0
		a.CreatedAt = &alertTime
		alerts = append(alerts, a)
	}

	WriteJSON(w, http.StatusOK, alerts)
}

// AcknowledgeAlertHandler handles POST /api/v1/alerts/{alert_id}/acknowledge
func AcknowledgeAlertHandler(w http.ResponseWriter, r *http.Request) {
	alertID := r.PathValue("alert_id")
	if alertID == "" {
		WriteError(w, http.StatusBadRequest, "Missing alert ID")
		return
	}

	// Update the alert to acknowledged = 1
	res, err := db.DB.Exec("UPDATE alerts SET acknowledged = 1 WHERE id = ?", alertID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update alert status")
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		WriteError(w, http.StatusNotFound, "Alert not found")
		return
	}

	// Get the updated alert to return
	var a models.AlertResponse
	var ruleID sql.NullString
	var ackVal int
	var alertTime time.Time
	err = db.DB.QueryRow(`
		SELECT id, rule_id, severity, message, acknowledged, created_at
		FROM alerts
		WHERE id = ?
	`, alertID).Scan(&a.ID, &ruleID, &a.Severity, &a.Message, &ackVal, &alertTime)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch updated alert")
		return
	}

	if ruleID.Valid {
		a.RuleID = &ruleID.String
	}
	a.Acknowledged = ackVal != 0
	a.CreatedAt = &alertTime

	WriteJSON(w, http.StatusOK, a)
}

// ListRulesHandler handles GET /api/v1/alert-rules
func ListRulesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
		SELECT id, metric_type, threshold, operator, duration_seconds, enabled, created_at
		FROM alert_rules
		ORDER BY created_at DESC
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to load rules")
		return
	}
	defer rows.Close()

	rules := []models.AlertRuleResponse{}
	for rows.Next() {
		var r models.AlertRuleResponse
		var enabledVal int
		var ruleTime time.Time
		err := rows.Scan(&r.ID, &r.MetricType, &r.Threshold, &r.Operator, &r.DurationSeconds, &enabledVal, &ruleTime)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Error reading rules")
			return
		}
		r.Enabled = enabledVal != 0
		r.CreatedAt = &ruleTime
		rules = append(rules, r)
	}

	WriteJSON(w, http.StatusOK, rules)
}

// CreateRuleHandler handles POST /api/v1/alert-rules
func CreateRuleHandler(w http.ResponseWriter, r *http.Request) {
	var body models.AlertRuleCreate
	if err := ParseJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	nowStr := now.Format("2006-01-02 15:04:05")

	enabledVal := 0
	if body.Enabled {
		enabledVal = 1
	}

	_, err := db.DB.Exec(`
		INSERT INTO alert_rules (id, metric_type, threshold, operator, duration_seconds, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, body.MetricType, body.Threshold, body.Operator, body.DurationSeconds, enabledVal, nowStr)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create rule")
		return
	}

	WriteJSON(w, http.StatusCreated, models.AlertRuleResponse{
		ID:              id,
		MetricType:      body.MetricType,
		Threshold:       body.Threshold,
		Operator:        body.Operator,
		DurationSeconds: body.DurationSeconds,
		Enabled:         body.Enabled,
		CreatedAt:       &now,
	})
}

// UpdateRuleHandler handles PUT /api/v1/alert-rules/{rule_id}
func UpdateRuleHandler(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("rule_id")
	if ruleID == "" {
		WriteError(w, http.StatusBadRequest, "Missing rule ID")
		return
	}

	var body models.AlertRuleUpdate
	if err := ParseJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	// Update existing record
	var currentThreshold float64
	var currentOperator string
	var currentDuration int
	var currentEnabled int
	err := db.DB.QueryRow(`
		SELECT threshold, operator, duration_seconds, enabled
		FROM alert_rules WHERE id = ?
	`, ruleID).Scan(&currentThreshold, &currentOperator, &currentDuration, &currentEnabled)

	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "Rule not found")
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if body.Threshold != nil {
		currentThreshold = *body.Threshold
	}
	if body.Operator != nil {
		currentOperator = *body.Operator
	}
	if body.DurationSeconds != nil {
		currentDuration = *body.DurationSeconds
	}
	if body.Enabled != nil {
		if *body.Enabled {
			currentEnabled = 1
		} else {
			currentEnabled = 0
		}
	}

	_, err = db.DB.Exec(`
		UPDATE alert_rules
		SET threshold = ?, operator = ?, duration_seconds = ?, enabled = ?
		WHERE id = ?
	`, currentThreshold, currentOperator, currentDuration, currentEnabled, ruleID)

	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	var resp models.AlertRuleResponse
	var ruleTime time.Time
	err = db.DB.QueryRow(`
		SELECT id, metric_type, threshold, operator, duration_seconds, enabled, created_at
		FROM alert_rules WHERE id = ?
	`, ruleID).Scan(&resp.ID, &resp.MetricType, &resp.Threshold, &resp.Operator, &resp.DurationSeconds, &currentEnabled, &ruleTime)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to load updated rule")
		return
	}

	resp.Enabled = currentEnabled != 0
	resp.CreatedAt = &ruleTime

	WriteJSON(w, http.StatusOK, resp)
}

// DeleteRuleHandler handles DELETE /api/v1/alert-rules/{rule_id}
func DeleteRuleHandler(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("rule_id")
	if ruleID == "" {
		WriteError(w, http.StatusBadRequest, "Missing rule ID")
		return
	}

	res, err := db.DB.Exec("DELETE FROM alert_rules WHERE id = ?", ruleID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		WriteError(w, http.StatusNotFound, "Rule not found")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Rule deleted"})
}
