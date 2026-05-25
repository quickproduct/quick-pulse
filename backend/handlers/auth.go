package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"quickpulse/backend/auth"
	"quickpulse/backend/db"
	"quickpulse/backend/models"
)

// LoginHandler handles POST /api/v1/auth/login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var id, email, hashedPassword, role string
	var isActive int
	err := db.DB.QueryRow(
		"SELECT id, email, hashed_password, role, is_active FROM users WHERE email = ?",
		req.Email,
	).Scan(&id, &email, &hashedPassword, &role, &isActive)

	if err == sql.ErrNoRows {
		WriteError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if isActive == 0 {
		WriteError(w, http.StatusForbidden, "User account is inactive")
		return
	}

	if !auth.CheckPasswordHash(req.Password, hashedPassword) {
		WriteError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	accessToken, err := auth.GenerateAccessToken(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	// Save session in SQLite
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = db.DB.Exec(
		"INSERT INTO sessions (id, user_id, refresh_token, expires_at) VALUES (?, ?, ?, ?)",
		sessionID, id, refreshToken, expiresAt,
	)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to store session")
		return
	}

	WriteJSON(w, http.StatusOK, models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
	})
}

// LogoutHandler handles POST /api/v1/auth/logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	_, _ = db.DB.Exec("DELETE FROM sessions WHERE refresh_token = ?", req.RefreshToken)

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
}

// RefreshHandler handles POST /api/v1/auth/refresh
func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	claims, err := auth.VerifyToken(req.RefreshToken, "refresh")
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	var sessionExists bool
	var expiresAtStr string
	err = db.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM sessions WHERE refresh_token = ?), expires_at FROM sessions WHERE refresh_token = ?",
		req.RefreshToken, req.RefreshToken,
	).Scan(&sessionExists, &expiresAtStr)

	if err != nil || !sessionExists {
		WriteError(w, http.StatusUnauthorized, "Session not found")
		return
	}

	expiresAt, err := time.Parse("2006-01-02 15:04:05", expiresAtStr)
	if err != nil || time.Now().After(expiresAt) {
		_, _ = db.DB.Exec("DELETE FROM sessions WHERE refresh_token = ?", req.RefreshToken)
		WriteError(w, http.StatusUnauthorized, "Refresh token expired")
		return
	}

	newAccessToken, err := auth.GenerateAccessToken(claims.Sub)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	newRefreshToken, err := auth.GenerateRefreshToken(claims.Sub)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	// Update session with new refresh token
	newExpiresAt := time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = db.DB.Exec(
		"UPDATE sessions SET refresh_token = ?, expires_at = ? WHERE refresh_token = ?",
		newRefreshToken, newExpiresAt, req.RefreshToken,
	)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update session")
		return
	}

	WriteJSON(w, http.StatusOK, models.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "bearer",
	})
}

// MeHandler handles GET /api/v1/me
func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var id, email, role, createdAtStr string
	var isActive int
	err := db.DB.QueryRow(
		"SELECT id, email, role, is_active, created_at FROM users WHERE id = ?",
		userID,
	).Scan(&id, &email, &role, &isActive, &createdAtStr)

	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	createdAt, _ := time.Parse("2006-01-02 15:04:05", createdAtStr)

	WriteJSON(w, http.StatusOK, models.UserResponse{
		ID:        id,
		Email:     email,
		Role:      role,
		IsActive:  isActive != 0,
		CreatedAt: &createdAt,
	})
}

// ChangePasswordHandler handles PUT /api/v1/auth/password
func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.ChangePasswordRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var hashedPassword string
	err := db.DB.QueryRow("SELECT hashed_password FROM users WHERE id = ?", userID).Scan(&hashedPassword)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if !auth.CheckPasswordHash(req.CurrentPassword, hashedPassword) {
		WriteError(w, http.StatusBadRequest, "Invalid current password")
		return
	}

	newHashed, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to hash new password")
		return
	}

	_, err = db.DB.Exec("UPDATE users SET hashed_password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newHashed, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Password changed"})
}

// RegisterHandler handles POST /api/v1/auth/register (if allowed)
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	allowReg := os.Getenv("ALLOW_REGISTRATION") == "true"
	if !allowReg {
		WriteError(w, http.StatusForbidden, "Open registration is disabled. Contact an admin for an invite.")
		return
	}

	var req models.RegisterRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", req.Email).Scan(&exists)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if exists {
		WriteError(w, http.StatusConflict, "Email already registered")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	id := uuid.New().String()
	_, err = db.DB.Exec(
		"INSERT INTO users (id, email, hashed_password, role, is_active) VALUES (?, ?, ?, 'admin', 1)",
		id, req.Email, hashed,
	)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	now := time.Now().UTC()
	WriteJSON(w, http.StatusCreated, models.UserResponse{
		ID:        id,
		Email:     req.Email,
		Role:      "admin",
		IsActive:  true,
		CreatedAt: &now,
	})
}
