package main

import (
	"context"
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"quickpulse/backend/db"
	"quickpulse/backend/handlers"
	logsservice "quickpulse/backend/logs/service"
	"quickpulse/backend/workers"
	"quickpulse/backend/ws"
)

//go:embed all:frontend/build
var staticFS embed.FS

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// 1. Initialize SQLite Database
	db.InitDB()

	// 2. Start WebSocket Heartbeat & Workers
	ws.Manager.StartHeartbeat()
	workers.StartMetricsWorker()
	workers.StartEventsWorker()
	workers.StartMetricsJanitorWorker()

	// 2b. Start the logs module. We use Background() as the parent — the
	// process lifetime is the service lifetime. Failures here aren't fatal;
	// the logs endpoints just return 503 until the next restart.
	hostName, _ := os.Hostname()
	if svc, err := logsservice.Start(context.Background(), db.DB, hostName); err != nil {
		log.Printf("[logs] failed to start: %v — logs endpoints will return 503", err)
	} else {
		handlers.LogsService = svc
		log.Printf("[logs] service started")
	}

	// 3. Define HTTP Mux with Go 1.22 routing enhancement
	mux := http.NewServeMux()

	// --- Health Check ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// --- Auth API ---
	mux.HandleFunc("POST /api/v1/auth/login", handlers.LoginHandler)
	mux.HandleFunc("POST /api/v1/auth/logout", handlers.LogoutHandler)
	mux.HandleFunc("POST /api/v1/auth/refresh", handlers.RefreshHandler)
	mux.HandleFunc("GET /api/v1/me", handlers.AuthMiddleware(handlers.MeHandler))
	mux.HandleFunc("PUT /api/v1/auth/password", handlers.AuthMiddleware(handlers.ChangePasswordHandler))
	mux.HandleFunc("POST /api/v1/auth/register", handlers.RegisterHandler)

	// --- Metrics API ---
	mux.HandleFunc("GET /api/v1/metrics/summary", handlers.AuthMiddleware(handlers.GetMetricsSummaryHandler))
	mux.HandleFunc("GET /api/v1/metrics/history", handlers.AuthMiddleware(handlers.GetMetricsHistoryHandler))

	// --- Dashboard API ---
	mux.HandleFunc("GET /api/v1/dashboard", handlers.AuthMiddleware(handlers.GetDashboardHandler))

	// --- Events API ---
	mux.HandleFunc("GET /api/v1/events", handlers.AuthMiddleware(handlers.GetEventsHandler))

	// --- Containers API ---
	mux.HandleFunc("GET /api/v1/containers", handlers.AuthMiddleware(handlers.ListContainersHandler))
	mux.HandleFunc("GET /api/v1/containers/{id}", handlers.AuthMiddleware(handlers.InspectContainerHandler))
	mux.HandleFunc("POST /api/v1/containers/{id}/start", handlers.AuthMiddleware(handlers.StartContainerHandler))
	mux.HandleFunc("POST /api/v1/containers/{id}/stop", handlers.AuthMiddleware(handlers.StopContainerHandler))
	mux.HandleFunc("POST /api/v1/containers/{id}/restart", handlers.AuthMiddleware(handlers.RestartContainerHandler))
	mux.HandleFunc("GET /api/v1/containers/{id}/logs", handlers.AuthMiddleware(handlers.GetContainerLogsHandler))

	// --- Stacks API ---
	mux.HandleFunc("GET /api/v1/stacks", handlers.AuthMiddleware(handlers.ListStacksHandler))
	mux.HandleFunc("POST /api/v1/stacks", handlers.AuthMiddleware(handlers.CreateStackHandler))
	mux.HandleFunc("GET /api/v1/stacks/{name}", handlers.AuthMiddleware(handlers.GetStackHandler))
	mux.HandleFunc("POST /api/v1/stacks/{name}/start", handlers.AuthMiddleware(handlers.StartStackHandler))
	mux.HandleFunc("POST /api/v1/stacks/{name}/stop", handlers.AuthMiddleware(handlers.StopStackHandler))
	mux.HandleFunc("POST /api/v1/stacks/{name}/restart", handlers.AuthMiddleware(handlers.RestartStackHandler))
	mux.HandleFunc("GET /api/v1/stacks/{name}/config", handlers.AuthMiddleware(handlers.GetStackConfigHandler))
	mux.HandleFunc("POST /api/v1/stacks/{name}/config", handlers.AuthMiddleware(handlers.SaveStackConfigHandler))
	mux.HandleFunc("POST /api/v1/stacks/{name}/deploy", handlers.AuthMiddleware(handlers.DeployStackHandler))

	// --- Alerts API ---
	mux.HandleFunc("GET /api/v1/alerts", handlers.AuthMiddleware(handlers.ListAlertsHandler))
	mux.HandleFunc("POST /api/v1/alerts/{alert_id}/acknowledge", handlers.AuthMiddleware(handlers.AcknowledgeAlertHandler))
	mux.HandleFunc("GET /api/v1/alert-rules", handlers.AuthMiddleware(handlers.ListRulesHandler))
	mux.HandleFunc("POST /api/v1/alert-rules", handlers.AuthMiddleware(handlers.CreateRuleHandler))
	mux.HandleFunc("PUT /api/v1/alert-rules/{rule_id}", handlers.AuthMiddleware(handlers.UpdateRuleHandler))
	mux.HandleFunc("DELETE /api/v1/alert-rules/{rule_id}", handlers.AuthMiddleware(handlers.DeleteRuleHandler))

	// --- Kubernetes API ---
	mux.HandleFunc("GET /api/v1/kubernetes/contexts", handlers.AuthMiddleware(handlers.K8sContextsHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/overview", handlers.AuthMiddleware(handlers.K8sOverviewHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/nodes", handlers.AuthMiddleware(handlers.K8sNodesHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/pods", handlers.AuthMiddleware(handlers.K8sPodsHandler))
	mux.HandleFunc("DELETE /api/v1/kubernetes/pods/{namespace}/{name}", handlers.AuthMiddleware(handlers.DeletePodHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/deployments", handlers.AuthMiddleware(handlers.K8sDeploymentsHandler))
	mux.HandleFunc("POST /api/v1/kubernetes/deployments/{namespace}/{name}/scale", handlers.AuthMiddleware(handlers.ScaleDeploymentHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/services", handlers.AuthMiddleware(handlers.K8sServicesHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/namespaces", handlers.AuthMiddleware(handlers.K8sNamespacesHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/events", handlers.AuthMiddleware(handlers.K8sEventsHandler))
	mux.HandleFunc("GET /api/v1/kubernetes/pods/{namespace}/{pod_name}/logs", handlers.AuthMiddleware(handlers.GetK8sPodLogsHandler))

	// --- Logs API (centralized search across all sources) ---
	mux.HandleFunc("GET /api/v1/logs", handlers.AuthMiddleware(handlers.SearchLogsHandler))
	mux.HandleFunc("GET /api/v1/logs/sources", handlers.AuthMiddleware(handlers.SourcesHandler))
	mux.HandleFunc("GET /api/v1/logs/stats", handlers.AuthMiddleware(handlers.StatsHandler))
	mux.HandleFunc("GET /api/v1/logs/export", handlers.AuthMiddleware(handlers.ExportLogsHandler))
	mux.HandleFunc("GET /api/v1/logs/settings", handlers.AuthMiddleware(handlers.SettingsGetHandler))
	mux.HandleFunc("PUT /api/v1/logs/settings", handlers.AdminMiddleware(handlers.SettingsPutHandler))
	mux.HandleFunc("GET /api/v1/logs/{id}", handlers.AuthMiddleware(handlers.GetLogHandler))

	// --- WebSockets ---
	mux.HandleFunc("GET /ws/metrics", handlers.HandleWSChannel("metrics"))
	mux.HandleFunc("GET /ws/container-status", handlers.HandleWSChannel("container-status"))
	mux.HandleFunc("GET /ws/events", handlers.HandleWSChannel("events"))
	mux.HandleFunc("GET /ws/logs/stream", handlers.HandleWSLogsStream)
	mux.HandleFunc("GET /ws/logs/{container_id}", handlers.HandleWSLogs)
	mux.HandleFunc("GET /ws/kubernetes/logs/{namespace}/{pod_name}", handlers.HandleWSK8sLogs)
	mux.HandleFunc("GET /ws/containers/{id}/terminal", handlers.HandleWSContainerTerminal)

	// --- Static Frontend Files Embedded ---
	fSys, err := fs.Sub(staticFS, "frontend/build")
	if err != nil {
		log.Fatalf("Failed to locate embedded frontend files: %v", err)
	}
	fileServer := http.FileServer(http.FS(fSys))

	// Catch-all handler for serving SPA static files or routing to index.html
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Ensure we don't handle API/WS routes that missed path matches (e.g. 404s on APIs)
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// Check if file exists in the embedded filesystem
		_, err := fSys.Open(cleanPath)
		if err != nil {
			// If file does not exist, serve index.html (client-side routing fallback)
			indexFile, err := fSys.Open("index.html")
			if err != nil {
				http.Error(w, "Static asset index.html not found", http.StatusNotFound)
				return
			}
			defer indexFile.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, indexFile)
			return
		}

		// File exists, serve it
		fileServer.ServeHTTP(w, r)
	})

	// 4. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	addr := ":" + port

	log.Printf("Starting consolidated QuickPulse backend on %s", addr)
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatalf("Failed to run HTTP server: %v", err)
	}
}
