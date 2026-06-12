package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"quickpulse/backend/internal/auth"
	"quickpulse/backend/internal/db"
)

type contextKey string

const UserIDKey contextKey = "userId"

// WriteJSON sends a JSON response with status code
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// WriteError sends a JSON error detail payload matching FastAPI format
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"detail": message})
}

// ParseJSON decodes JSON body from HTTP request
func ParseJSON(r *http.Request, data interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(data)
}

// AuthMiddleware verifies the JWT access token and adds User ID to context
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}

		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			WriteError(w, http.StatusUnauthorized, "Missing authorization header or token query parameter")
			return
		}

		claims, err := auth.VerifyToken(token, "access")
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.Sub)
		next(w, r.WithContext(ctx))
	}
}

// AdminMiddleware verifies the user is authenticated and has the admin role
func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDKey).(string)
		if !ok {
			WriteError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		var role string
		var isActive int
		err := db.DB.QueryRow("SELECT role, is_active FROM users WHERE id = ?", userID).Scan(&role, &isActive)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "User not found")
			return
		}

		if isActive == 0 {
			WriteError(w, http.StatusForbidden, "User account is inactive")
			return
		}

		if role != "admin" {
			WriteError(w, http.StatusForbidden, "Admin permission required")
			return
		}

		next(w, r)
	})
}
