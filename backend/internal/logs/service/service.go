// Package service is the orchestrator for the logs module. It owns the
// ingest pipeline, the broker, and the long-lived collector goroutines.
// Sub-packages (collector, parser, store, pubsub) and the top-level `logs`
// data-types package are all imported here — but nothing under `logs/`
// imports back into `service`, so the dependency graph stays acyclic.
package service

import (
	"context"
	"database/sql"
	"go.uber.org/zap"
	"os"
	"strconv"
	"time"

	"quickpulse/backend/internal/logs"
	"quickpulse/backend/internal/logs/collector"
	"quickpulse/backend/internal/logs/parser"
	"quickpulse/backend/internal/logs/pubsub"
	"quickpulse/backend/internal/logs/store"
)

// Service is the public face of the logs module — main.go constructs one
// and the HTTP handlers reach into it for store/broker/ingest access.
type Service struct {
	Store    *store.SQLiteStore
	Broker   *pubsub.Broker
	Ingest   *Ingest
	Reg      *collector.Registry
	Settings logs.Settings
}

// brokerAdapter bridges Broker (which takes []logs.Entry) to ingest's
// BatchPublisher interface — they're structurally identical but the
// interface lives in this package so the broker doesn't have to import it.
type brokerAdapter struct {
	b *pubsub.Broker
}

func (a *brokerAdapter) Publish(batch []logs.Entry) { a.b.Publish(batch) }

// Start wires everything together and launches background goroutines. It
// returns immediately; the parent process should cancel `ctx` for graceful
// shutdown. Pass `db` (the existing SQLite handle) so we don't open a
// second pool.
func Start(ctx context.Context, db *sql.DB, hostName string) (*Service, error) {
	st, err := store.New(db)
	if err != nil {
		return nil, err
	}

	settings, err := st.LoadSettings(ctx)
	if err != nil {
		zap.L().Sugar().Infof("[logs] settings load failed, using defaults: %v", err)
		settings = logs.DefaultSettings()
	}

	broker := pubsub.NewBroker(intEnv("LOGS_MAX_SUBSCRIBERS", 16))
	adapter := &brokerAdapter{b: broker}

	ingest := NewIngest(ctx, st, adapter, settings,
		intEnv("LOGS_INGEST_BUFFER", 1024),
		intEnv("LOGS_BATCH_SIZE", 200),
		durEnv("LOGS_BATCH_INTERVAL_MS", 250*time.Millisecond),
	)

	reg := collector.NewRegistry(intEnv("LOGS_MAX_STREAMS", 32))
	joiner := parser.NewMultilineJoiner()
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	submitter := submitterFn(func(e logs.Entry) { ingest.Submit(e) })

	// Docker collector: tolerate failure (some hosts run K8s-only).
	if d, err := collector.NewDocker(reg, joiner, submitter, hostName, env); err != nil {
		zap.L().Sugar().Infof("[logs/docker] disabled: %v", err)
	} else {
		go d.Run(ctx)
		zap.L().Sugar().Infof("[logs/docker] started")
	}

	// K8s collector: same — kubeconfig may be absent.
	k := collector.NewK8s(reg, joiner, submitter, env)
	go k.Run(ctx)
	zap.L().Sugar().Infof("[logs/k8s] started")

	svc := &Service{
		Store:    st,
		Broker:   broker,
		Ingest:   ingest,
		Reg:      reg,
		Settings: settings,
	}

	// Janitor: prune by retention every 5 minutes.
	go svc.runJanitor(ctx)

	return svc, nil
}

// submitterFn lets us pass a closure as a collector.Submitter without
// declaring a struct type per submitter.
type submitterFn func(logs.Entry)

// Submit satisfies the collector.Submitter interface.
func (f submitterFn) Submit(e logs.Entry) { f(e) }

// ApplySettings updates the running config (sample ratios, retention).
// Called from the settings HTTP handler.
func (s *Service) ApplySettings(ctx context.Context, st logs.Settings) error {
	if err := s.Store.SaveSettings(ctx, st); err != nil {
		return err
	}
	s.Settings = st
	s.Ingest.UpdateSettings(st)
	return nil
}

// runJanitor is the retention loop. Runs every 5 minutes, prunes by both
// age and size, and checkpoints the WAL hourly.
func (s *Service) runJanitor(ctx context.Context) {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	checkpointTick := time.NewTicker(time.Hour)
	defer checkpointTick.Stop()

	runOnce := func() {
		latest, err := s.Store.LoadSettings(ctx)
		if err == nil {
			s.Settings = latest
		}
		jctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		n, err := s.Store.Vacuum(jctx, s.Settings)
		if err != nil {
			zap.L().Sugar().Infof("[logs/janitor] vacuum: %v", err)
			return
		}
		if n > 0 {
			zap.L().Sugar().Infof("[logs/janitor] evicted %d rows", n)
		}
	}

	// First pass shortly after startup.
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
			runOnce()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runOnce()
		case <-checkpointTick.C:
			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if _, err := walCheckpoint(cctx); err != nil {
				zap.L().Sugar().Infof("[logs/janitor] wal_checkpoint: %v", err)
			}
			cancel()
		}
	}
}

// intEnv reads an integer env var, falling back to default.
func intEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// durEnv reads a duration-in-ms env var.
func durEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Millisecond
		}
	}
	return def
}
