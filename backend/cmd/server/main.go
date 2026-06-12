// Command server runs the consolidated QuickPulse backend: a single binary
// serving the embedded SvelteKit dashboard, the REST/WebSocket API, and the
// background workers (metrics, events, logs).
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"quickpulse/backend/internal/config"
	"quickpulse/backend/internal/db"
	"quickpulse/backend/internal/handlers"
	"quickpulse/backend/internal/logging"
	logsservice "quickpulse/backend/internal/logs/service"
	"quickpulse/backend/internal/router"
	"quickpulse/backend/internal/workers"
	"quickpulse/backend/internal/ws"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Env)
	zap.ReplaceGlobals(logger)
	defer func() { _ = logger.Sync() }()

	logger.Info("starting quickpulse",
		zap.String("env", cfg.Env),
		zap.String("port", cfg.Port),
		zap.String("db", cfg.DatabasePath),
	)

	// ── Persistence ─────────────────────────────────────────────────────────
	if err := db.InitDB(db.Config{
		Path:          cfg.DatabasePath,
		AdminEmail:    cfg.AdminEmail,
		AdminPassword: cfg.AdminPassword,
	}, logger); err != nil {
		logger.Fatal("init database", zap.Error(err))
	}

	// ── Background workers ────────────────────────────────────────────────────
	ws.Manager.StartHeartbeat()
	workers.StartMetricsWorker()
	workers.StartEventsWorker()
	workers.StartMetricsJanitorWorker()

	hostName, _ := os.Hostname()
	if svc, err := logsservice.Start(context.Background(), db.DB, hostName); err != nil {
		logger.Warn("logs service failed to start; logs endpoints will 503", zap.Error(err))
	} else {
		handlers.LogsService = svc
		logger.Info("logs service started")
	}

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
	logger.Info("shutdown complete")
}
