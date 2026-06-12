// Package router assembles the chi router with middleware and all routes.
package router

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"quickpulse/backend/internal/handlers"
	"quickpulse/backend/internal/httpx"
	"quickpulse/backend/internal/static"
)

// New builds the application router.
func New(logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(httpx.CORS)
	r.Use(httpx.RequestLogger(logger))

	auth := handlers.AuthMiddleware
	admin := handlers.AdminMiddleware

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// ── Auth ───────────────────────────────────────────────────────────────
	r.Post("/api/v1/auth/login", handlers.LoginHandler)
	r.Post("/api/v1/auth/logout", handlers.LogoutHandler)
	r.Post("/api/v1/auth/refresh", handlers.RefreshHandler)
	r.Post("/api/v1/auth/register", handlers.RegisterHandler)
	r.Get("/api/v1/me", auth(handlers.MeHandler))
	r.Put("/api/v1/auth/password", auth(handlers.ChangePasswordHandler))

	// ── Metrics & dashboard ────────────────────────────────────────────────
	r.Get("/api/v1/metrics/summary", auth(handlers.GetMetricsSummaryHandler))
	r.Get("/api/v1/metrics/history", auth(handlers.GetMetricsHistoryHandler))
	r.Get("/api/v1/dashboard", auth(handlers.GetDashboardHandler))
	r.Get("/api/v1/events", auth(handlers.GetEventsHandler))

	// ── Containers ─────────────────────────────────────────────────────────
	r.Get("/api/v1/containers", auth(handlers.ListContainersHandler))
	r.Get("/api/v1/containers/{id}", auth(handlers.InspectContainerHandler))
	r.Post("/api/v1/containers/{id}/start", auth(handlers.StartContainerHandler))
	r.Post("/api/v1/containers/{id}/stop", auth(handlers.StopContainerHandler))
	r.Post("/api/v1/containers/{id}/restart", auth(handlers.RestartContainerHandler))
	r.Get("/api/v1/containers/{id}/logs", auth(handlers.GetContainerLogsHandler))

	// ── Stacks ─────────────────────────────────────────────────────────────
	r.Get("/api/v1/stacks", auth(handlers.ListStacksHandler))
	r.Post("/api/v1/stacks", auth(handlers.CreateStackHandler))
	r.Get("/api/v1/stacks/{name}", auth(handlers.GetStackHandler))
	r.Post("/api/v1/stacks/{name}/start", auth(handlers.StartStackHandler))
	r.Post("/api/v1/stacks/{name}/stop", auth(handlers.StopStackHandler))
	r.Post("/api/v1/stacks/{name}/restart", auth(handlers.RestartStackHandler))
	r.Get("/api/v1/stacks/{name}/config", auth(handlers.GetStackConfigHandler))
	r.Post("/api/v1/stacks/{name}/config", auth(handlers.SaveStackConfigHandler))
	r.Post("/api/v1/stacks/{name}/deploy", auth(handlers.DeployStackHandler))

	// ── Alerts ─────────────────────────────────────────────────────────────
	r.Get("/api/v1/alerts", auth(handlers.ListAlertsHandler))
	r.Post("/api/v1/alerts/{alert_id}/acknowledge", auth(handlers.AcknowledgeAlertHandler))
	r.Get("/api/v1/alert-rules", auth(handlers.ListRulesHandler))
	r.Post("/api/v1/alert-rules", auth(handlers.CreateRuleHandler))
	r.Put("/api/v1/alert-rules/{rule_id}", auth(handlers.UpdateRuleHandler))
	r.Delete("/api/v1/alert-rules/{rule_id}", auth(handlers.DeleteRuleHandler))

	// ── Kubernetes ─────────────────────────────────────────────────────────
	r.Get("/api/v1/kubernetes/contexts", auth(handlers.K8sContextsHandler))
	r.Get("/api/v1/kubernetes/overview", auth(handlers.K8sOverviewHandler))
	r.Get("/api/v1/kubernetes/nodes", auth(handlers.K8sNodesHandler))
	r.Get("/api/v1/kubernetes/pods", auth(handlers.K8sPodsHandler))
	r.Delete("/api/v1/kubernetes/pods/{namespace}/{name}", auth(handlers.DeletePodHandler))
	r.Get("/api/v1/kubernetes/deployments", auth(handlers.K8sDeploymentsHandler))
	r.Post("/api/v1/kubernetes/deployments/{namespace}/{name}/scale", auth(handlers.ScaleDeploymentHandler))
	r.Get("/api/v1/kubernetes/services", auth(handlers.K8sServicesHandler))
	r.Get("/api/v1/kubernetes/namespaces", auth(handlers.K8sNamespacesHandler))
	r.Get("/api/v1/kubernetes/events", auth(handlers.K8sEventsHandler))
	r.Get("/api/v1/kubernetes/pods/{namespace}/{pod_name}/logs", auth(handlers.GetK8sPodLogsHandler))

	// ── Logs (centralized search) ──────────────────────────────────────────
	r.Get("/api/v1/logs", auth(handlers.SearchLogsHandler))
	r.Get("/api/v1/logs/sources", auth(handlers.SourcesHandler))
	r.Get("/api/v1/logs/stats", auth(handlers.StatsHandler))
	r.Get("/api/v1/logs/export", auth(handlers.ExportLogsHandler))
	r.Get("/api/v1/logs/settings", auth(handlers.SettingsGetHandler))
	r.Put("/api/v1/logs/settings", admin(handlers.SettingsPutHandler))
	r.Get("/api/v1/logs/{id}", auth(handlers.GetLogHandler))

	// ── WebSockets ─────────────────────────────────────────────────────────
	r.Get("/ws/metrics", handlers.HandleWSChannel("metrics"))
	r.Get("/ws/container-status", handlers.HandleWSChannel("container-status"))
	r.Get("/ws/events", handlers.HandleWSChannel("events"))
	r.Get("/ws/logs/stream", handlers.HandleWSLogsStream)
	r.Get("/ws/logs/{container_id}", handlers.HandleWSLogs)
	r.Get("/ws/kubernetes/logs/{namespace}/{pod_name}", handlers.HandleWSK8sLogs)
	r.Get("/ws/containers/{id}/terminal", handlers.HandleWSContainerTerminal)

	// ── SPA fallback ───────────────────────────────────────────────────────
	spa := static.Handler()
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			http.NotFound(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	})

	return r
}
