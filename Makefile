# ============================================================================
# QuickPulse Makefile - Simplified Management
# ============================================================================

.PHONY: help up down logs ps migrate seed clean test lint

# Docker Compose command
COMPOSE := docker compose

help:
	@echo "QuickPulse Management Commands"
	@echo ""
	@echo "Main Commands:"
	@echo "  make up          Start all services (detached)"
	@echo "  make down        Stop all services"
	@echo "  make logs        Stream all service logs"
	@echo "  make ps          Show running containers"
	@echo "  make clean       Stop services and remove volumes"
	@echo ""
	@echo "Application Commands:"
	@echo "  make migrate     Run database migrations"
	@echo "  make seed        Seed default admin user"
	@echo "  make test        Run backend tests"
	@echo "  make lint        Run backend linter"
	@echo ""

up:
	@echo "🚀 Starting QuickPulse..."
	@$(COMPOSE) up -d
	@echo "✅ QuickPulse is running"

down:
	@echo "🛑 Stopping QuickPulse..."
	@$(COMPOSE) down
	@echo "✅ QuickPulse stopped"

logs:
	@$(COMPOSE) logs -f

ps:
	@$(COMPOSE) ps

migrate:
	@echo "🔄 Migrations are run automatically by the Go binary on startup."

seed:
	@echo "🌱 Seeding is handled automatically by the Go binary on startup."

clean:
	@echo "🧹 Cleaning up..."
	@$(COMPOSE) down -v
	@echo "✅ Cleanup complete"

test:
	@echo "🧪 Running backend unit tests..."
	cd backend && go test ./...

lint:
	@echo "🔍 Running Go vet and fmt checks..."
	cd backend && go vet ./... && go fmt ./...
