# QuickPulse Comprehensive Technical Guide

Welcome to the QuickPulse technical guide. QuickPulse is a self-hosted Docker and VPS management dashboard designed for high efficiency, developer convenience, and low resource overhead (runs within a 64MB RAM container).

---

## Technical Architecture

```mermaid
graph TD
    Client[SvelteKit Frontend (SPA)] -->|REST API / HTTP| GoServer[Go Backend Server]
    Client -->|WebSockets| GoServer
    GoServer -->|gopsutil| LocalOS[Host OS Metrics]
    GoServer -->|Docker SDK| DockerDaemon[Docker Daemon / Socket]
    GoServer -->|client-go| K8s[Kubernetes API Server]
    GoServer -->|modernc.org/sqlite| SQLite[(SQLite Database)]
```

### 1. Go Backend Server
The server-side application is written in Go 1.25+ and uses standard library routing enhanced by Go 1.22 path matching.
- **Embedded Frontend**: During compile-time, the SvelteKit production build is embedded directly into the Go binary using `go:embed`. The Go server acts as a single cohesive unit, serving both the REST APIs and static UI assets.
- **WebSocket Engine**: Implements bidirectional metrics and logging pipelines via `gorilla/websocket`.
- **System Metrics**: Periodically queries CPU, memory, disk, network, and load stats using the `shirou/gopsutil/v3` library.

### 2. SQLite Database Configuration
QuickPulse uses SQLite (`modernc.org/sqlite`, a CGO-free compiler-translated SQLite library) for settings, authentication, metrics history, alerts, and container events.
To maintain high performance and low lock contention:
- **Write-Ahead Logging (WAL) Mode**: Enabled using `PRAGMA journal_mode=WAL;`. This allows concurrent reads while writes are active.
- **Synchronous Normal**: Configured with `PRAGMA synchronous=NORMAL;` to reduce disk sync flushes.
- **Incremental Vacuum**: Database pages freed by the metrics cleanup worker are released back to the host OS incrementally with `PRAGMA auto_vacuum = INCREMENTAL;` and periodic `PRAGMA incremental_vacuum(50)` calls.
- **Metrics Janitor**: A background worker runs every 12 hours to prune high-resolution metrics older than 30 days and downsamples metrics older than 24 hours into 1-hour average buckets.

### 3. Model Context Protocol (MCP) Server
QuickPulse exposes its admin features as AI-agent tools using the Model Context Protocol (MCP).
- Located in the `/mcp` directory.
- Built using `FastMCP` (Python).
- Exposes tools like container control, alert inspection, metrics history fetching, and Kubernetes pod management.
- Leverages API authentication to safely query the Go REST endpoints.

### 4. Svelte 5 Frontend
The user interface is built using SvelteKit compile-to-static adapter and Skeleton UI (TailwindCSS v4 compatible).
- Uses Svelte 5 Runes (`$state`, `$derived`, `$effect`) for reactive state binding.
- Communication with Go backend REST endpoints is managed via a centralized `apiFetch` client that handles automatic JWT token refreshing (using access/refresh rotation) on HTTP `401 Unauthorized` responses.

---

## Deployment & Setup

Refer to the root [README.md](../README.md) for quick-start Docker instructions.

### Environment Configuration
The application reads settings from the environment. Key variables include:
- `PORT` (Default: `8000`): Port Go backend serves on.
- `SQLITE_DB_PATH` (Default: `quickpulse.db`): Path to SQLite database.
- `JWT_SECRET_KEY` (Required): String key to sign access/refresh tokens.
- `ALLOW_REGISTRATION` (Default: `true`): Toggles open registration for administrators.
- `METRICS_RETENTION_DAYS` (Default: `30`): Number of days to store host metrics history.
