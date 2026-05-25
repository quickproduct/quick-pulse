# QuickPulse

A lightweight, self-hosted Docker and VPS control panel for developers.

[![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)
[![Svelte](https://img.shields.io/badge/Svelte-FF3E00?style=for-the-badge&logo=svelte&logoColor=white)](https://svelte.dev/)
[![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)

## 🚀 Quick Start

1. **Clone & Configure**:
   ```bash
   git clone https://github.com/your-org/quickpulse.git
   cd quickpulse
   cp .env.example .env
   ```

2. **Start Services**:
   ```bash
   docker compose up -d
   ```

3. **Login**:
   Open [http://localhost](http://localhost) and log in with the credentials set in your `.env` file (Default: `admin@quickpulse.local` / `admin`).

## 📖 Documentation

For comprehensive information, including non-technical overviews, technical architecture, and developer guides, please see:

👉 **[QuickPulse Comprehensive Guide](docs/GUIDE.md)**

## 🛠 Features

- Real-time VPS, Docker, & Kubernetes cluster monitoring
- Container & Pod lifecycle management
- Live container & Pod log streaming (with WebSockets)
- Docker Compose stack support (accurate service counting)
- Configurable alerts & metrics history
- Dark mode & JWT security
- Model Context Protocol (MCP) server for AI assistants

## 🏗 Project Structure

- `backend/`: Go backend application (serves APIs and hosts embedded static frontend)
- `frontend/`: SvelteKit user interface
- `docs/`: Centralized documentation
- `mcp/`: AI Agent integration (Model Context Protocol server)
- `docker-compose.yml`: Consolidated single-container deployment configuration

---

License: MIT
