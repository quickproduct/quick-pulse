# QuickPulse

A lightweight, self-hosted Docker and VPS control panel for developers.

[![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)
[![Svelte](https://img.shields.io/badge/Svelte-FF3E00?style=for-the-badge&logo=svelte&logoColor=white)](https://svelte.dev/)
[![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)

## Quick Start

1. **Clone and configure**:
   ```bash
   git clone https://github.com/quickproduct/quick-pulse.git
   cd quickpulse
   cp .env.example .env
   ```

2. **Start the app**:
   ```bash
   docker compose up -d
   ```

3. **Log in**:
   Open [http://localhost](http://localhost) and log in with the credentials set in `.env` (default: `admin@quickpulse.local` / `admin`).

QuickPulse runs as one container: the root `Dockerfile` builds the SvelteKit frontend, embeds it into the Go backend binary, and serves the API, WebSockets, and static UI from port `8000` inside the container.

## Documentation

For comprehensive information, including non-technical overviews, technical architecture, and developer guides, please see:

**[QuickPulse Comprehensive Guide](docs/GUIDE.md)**

## Features

- Real-time VPS, Docker, and Kubernetes cluster monitoring
- Container and pod lifecycle management
- Live container and pod log streaming with WebSockets
- Docker Compose stack support (accurate service counting)
- Configurable alerts and metrics history
- Dark mode and JWT security
- Model Context Protocol (MCP) server for AI assistants

## Project Structure

- `backend/`: Go backend application (serves APIs and hosts embedded static frontend)
- `frontend/`: SvelteKit user interface
- `docs/`: Centralized documentation
- `mcp/`: AI agent integration (Model Context Protocol server)
- `docker-compose.yml`: Consolidated single-container deployment configuration

---

License: MIT
