# QuickPulse: Resource-Optimized Self-Hosted Docker & VPS Control Panel

QuickPulse is an ultra-lightweight, self-hosted control panel designed for developers to monitor and manage Docker containers and VPS infrastructure in real-time.

---

## 1. Product Overview (Non-Technical)

QuickPulse provides a centralized dashboard to keep track of your servers and containers. Since the product's primary purpose is monitoring, it has been optimized to consume **minimal resources (CPU and memory)**, making it ideal to run on low-end VPS servers without taking up CPU/RAM needed by your actual applications.

### Key Features:
- **Real-time Monitoring**: Track CPU, Memory, Disk, and Network usage with live updates.
- **Docker Management**: Start, stop, restart, and inspect containers directly from the UI.
- **Live Logs**: Stream container logs in real-time with pause/resume functionality.
- **Compose Support**: Manage Docker Compose stacks effortlessly.
- **Alerting**: Get notified when system metrics exceed your defined thresholds.
- **Developer-First UI**: Clean, modern interface with dark mode and JWT-based security.
- **AI Agent Integration (MCP)**: Exposes QuickPulse admin capabilities as Model Context Protocol (MCP) tools for AI assistants (like Cursor, Claude, etc.).

---

## 2. System Architecture

QuickPulse is consolidated into a single Docker container containing a compiled Go application with embedded static frontend assets.

### Tech Stack:
- **Backend**: Go (Single compiled binary, highly optimized memory footprint)
- **Database**: Embedded SQLite (Runs in WAL mode for concurrent, fast reads and writes with minimal memory overhead)
- **Message Bus / WS Pub-Sub**: Replaced Redis with an in-memory thread-safe Go pub/sub hub.
- **Frontend**: SvelteKit + Tailwind CSS (Embedded directly inside the Go binary using `go:embed` and served by Go's HTTP multiplexer)
- **Infrastructure**: Built into a single container (`qp-app`), completely eliminating Nginx, PostgreSQL, and Redis containers.

---

## 3. Developer & Setup Guide

### Prerequisites:
- Docker and Docker Compose (v2.0+)
- Go 1.26+ (for local backend development)
- Node.js 20+ (for local frontend development)
- `make` (optional, for utility commands)

### Local Development Setup:
1. **Clone the Repository**:
   ```bash
   git clone https://github.com/your-org/quickpulse.git
   cd quickpulse
   ```

2. **Configure Environment**:
   ```bash
   cp .env.example .env
   # Edit .env to set your secrets and preferred ports
   ```

3. **Start the Application**:
   ```bash
   docker compose up -d
   ```

4. **Access the Application**:
   - Access URL: [http://localhost](http://localhost) (or your configured port `EXTERNAL_FRONTEND_PORT`)
   - Health Check: [http://localhost/health](http://localhost/health)

### Common Makefile Commands:
- `make up`: Build and start the container.
- `make down`: Stop the container.
- `make logs`: Stream container logs.
- `make test`: Run Go unit tests in the backend.
- `make lint`: Format and vet Go code.
- `make clean`: Stop services and clean up volumes.

---

## 4. Deployment & Infrastructure

### Single Container Configuration:
QuickPulse relies on a single `docker-compose.yml` config file. Resource limits are configured to be extremely low (e.g. `0.25` CPU and `64m` memory limit), but the Go binary typically uses less than **15MB of RAM** and near **0.00% idle CPU**.

#### Volume Mounts:
- `/var/run/docker.sock`: Mounted as read-only to allow container management.
- `qp-data`: Persists the SQLite database.
- `./logs`: Persists runtime application log files.

---

## 5. API & WebSocket Reference

### Core API Endpoints:
- `POST /api/v1/auth/login`: Authenticate and get JWT.
- `GET /api/v1/containers`: List all Docker containers.
- `GET /api/v1/metrics/summary`: Get current system metrics.
- `GET /api/v1/dashboard`: Retrieve combined dashboard overview.

### WebSocket Channels:
- `/ws/metrics`: Live system metrics stream.
- `/ws/logs/{container_id}`: Live log stream for a specific container.
- `/ws/events`: Real-time Docker event feed.

---

## 6. AI Agent Integration (MCP)

QuickPulse includes an integrated Model Context Protocol (MCP) server under `mcp/` written in Python. This allows models (like Claude or Cursor) to administer your VPS:
- Exposes tools to list/manage containers, compose stacks, read logs, check alerts, and verify system metrics.
- Configuration: Set `QP_API_URL` (defaults to `http://localhost:8000`), `QP_ADMIN_EMAIL`, and `QP_ADMIN_PASSWORD` to authenticate MCP queries.
