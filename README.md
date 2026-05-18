# QuickPulse

A lightweight, self-hosted Docker and VPS control panel for developers.

[![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)
[![Svelte](https://img.shields.io/badge/Svelte-FF3E00?style=for-the-badge&logo=svelte&logoColor=white)](https://svelte.dev/)
[![FastAPI](https://img.shields.io/badge/FastAPI-005571?style=for-the-badge&logo=fastapi)](https://fastapi.tiangolo.com/)

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

- Real-time VPS & Docker monitoring
- Container lifecycle management (Start/Stop/Restart)
- Live container log streaming
- Docker Compose stack support
- Configurable alerts & metrics history
- Dark mode & JWT security

## 🏗 Project Structure

- `backend/`: FastAPI application
- `frontend/`: SvelteKit user interface
- `docs/`: Centralized documentation
- `mcp/`: AI Agent integration
- `docker-compose.yml`: Consolidated deployment config

---

License: MIT
