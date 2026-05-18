# QuickPulse: Self-Hosted Docker & VPS Control Panel

QuickPulse is a lightweight, self-hosted control panel designed for developers to monitor and manage Docker containers and VPS infrastructure in real-time.

---

## 1. Product Overview (Non-Technical)

QuickPulse provides a centralized dashboard to keep track of your servers and containers. Whether you're running a single VPS or a cluster of Dockerized applications, QuickPulse gives you the visibility and control you need.

### Key Features:
- **Real-time Monitoring**: Track CPU, Memory, Disk, and Network usage with live updates.
- **Docker Management**: Start, stop, restart, and inspect containers directly from the UI.
- **Live Logs**: Stream container logs in real-time with pause/resume functionality.
- **Compose Support**: Manage Docker Compose stacks effortlessly.
- **Alerting**: Get notified when system metrics exceed your defined thresholds.
- **Developer-First UI**: Clean, modern interface with dark mode and JWT-based security.

---

## 2. System Architecture

QuickPulse is built with modern, high-performance technologies to ensure low latency and minimal resource footprint.

### Tech Stack:
- **Backend**: Python 3.12 + FastAPI (Asynchronous, High Performance)
- **Real-time**: WebSockets for live metrics, logs, and events.
- **Database**: PostgreSQL with TimescaleDB for efficient time-series metrics storage.
- **Caching**: Redis for session management and real-time pub/sub.
- **Frontend**: SvelteKit + Tailwind CSS (Fast, Reactive UI).
- **Infrastructure**: Docker & Docker Compose for easy, isolated deployment.

### Component Overview:
- `backend/`: FastAPI server handling API requests, WebSocket streams, and background workers.
- `frontend/`: SvelteKit application providing the user interface.
- `mcp/`: Model Context Protocol implementation for AI agent integration.
- `db`: PostgreSQL/TimescaleDB for persistent data.
- `redis`: Redis for caching and real-time message bus.

---

## 3. Developer Guide

### Prerequisites:
- Docker and Docker Compose (v2.0+)
- Python 3.12 (for local backend development)
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

3. **Start Infrastructure**:
   ```bash
   docker compose up -d
   ```

4. **Access the Application**:
   - Frontend: [http://localhost](http://localhost) (or your configured port)
   - Backend API: [http://localhost:8000](http://localhost:8000)
   - API Docs: [http://localhost:8000/docs](http://localhost:8000/docs)

### Common Commands:
- `make up`: Start all services.
- `make down`: Stop all services.
- `make logs`: View logs.
- `make migrate`: Run database migrations.
- `make seed`: Seed the default admin user.

---

## 4. Deployment & Infrastructure

### Docker Compose Setup:
QuickPulse uses a single `docker-compose.yml` file for all environments. Behavior is controlled via the `.env` file.

#### Environment Variables:
| Variable | Description |
|----------|-------------|
| `ENVIRONMENT` | `development`, `staging`, or `production` |
| `CONTAINER_PREFIX` | Prefix for container names (e.g., `qp`) |
| `JWT_SECRET_KEY` | Secret key for JWT token signing |
| `DB_PASSWORD` | Password for the PostgreSQL database |
| `EXTERNAL_FRONTEND_PORT` | Port to expose the frontend (default 80) |

### Production Best Practices:
1. **Security**: Change all default passwords and secrets in `.env`.
2. **Exclusion**: In production, do not expose the database or redis ports to the public internet.
3. **Reverse Proxy**: Use a reverse proxy (like Nginx, Traefik, or Caddy) with SSL/TLS in front of QuickPulse.
4. **Volumes**: Ensure persistent volumes (`qp-data`, `qp-redis`) are backed up regularly.

---

## 5. API & WebSocket Reference

### Core API Endpoints:
- `POST /api/v1/auth/login`: Authenticate and get JWT.
- `GET /api/v1/containers`: List all Docker containers.
- `GET /api/v1/metrics/summary`: Get current system metrics.

### WebSocket Channels:
- `/ws/metrics`: Live system metrics stream.
- `/ws/logs/{container_id}`: Live log stream for a specific container.
- `/ws/events`: Real-time Docker event feed.

---

*For detailed implementation history, refer to the project's internal changelogs.*
