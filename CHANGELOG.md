# Changelog

All notable changes to the QuickPulse project will be documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [1.0.0] - 2026-05-30

Initial release of consolidated production-grade QuickPulse.

### Added
- Multi-context Kubernetes support (overview status, pods management, nodes allocations, services listing).
- Docker Compose stacks editor and streaming logs deployer.
- Model Context Protocol (MCP) server for admin tool calling by AI assistants.
- SQLite indexing on metrics and alerts histories to optimize search and downsampling latency.
- Silent JWT token refreshing logic on frontend HTTP requests queue.

### Fixed
- Parameter names `{id}` matching bugs in alert update/delete and alert acknowledge routing.
- WebSocket authorization privilege gaps for deactivated accounts.
- Sanitized stack names across all routes to prevent path traversals.
- Fixed broken environment setup and validation bash scripts by recreating `/infra` configs.
