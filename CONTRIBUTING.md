# Contributing to QuickPulse

First off, thank you for considering contributing to QuickPulse! Contributions from developers like you are what make open-source projects sustainable and robust.

---

## Code of Conduct

We expect all contributors to adhere to respect, inclusive terminology, and collaboration. Please be welcoming and supportive.

---

## Local Development Setup

To configure the application locally:

### 1. Backend (Go)
1. Ensure you have **Go 1.25+** installed.
2. Initialize environment variables:
   ```bash
   cp .env.example .env
   ```
3. Run the Go server:
   ```bash
   cd backend
   go run main.go
   ```

### 2. Frontend (SvelteKit)
1. Ensure you have **Node.js 20+** installed.
2. Install Svelte dependencies:
   ```bash
   cd frontend
   npm install
   ```
3. Run SvelteKit dev server (configured to proxy `/api` requests to port `8000`):
   ```bash
   npm run dev
   ```

---

## Coding Guidelines

### 1. Go Style Guide
- All code must pass `go fmt` and `go vet` checks.
- Handle all errors explicitly. Do not ignore errors using blank identifier `_` unless explicitly documented why it is safe.
- Package layout should follow the clean architectural structure: handlers focus on HTTP, services/workers focus on background loops.

### 2. Svelte / TypeScript Style Guide
- Use Svelte 5 Runes (`$state`, `$derived`, `$effect`) for state reactive logic.
- Avoid utility placeholders or un-typed `any` bindings where TypeScript contracts can be declared.
- Format all files using Prettier standards.

---

## Submitting Pull Requests (PRs)

1. Fork the repository and create your branch from `main`.
2. Write unit tests for new logic where possible.
3. Verify that the build completes successfully:
   - Go: `cd backend && go test ./...`
   - Svelte: `cd frontend && npm run check && npm run build`
4. Follow commit naming conventions:
   - `feat: ...` for new features
   - `fix: ...` for bug fixes
   - `docs: ...` for documentation changes
   - `chore: ...` for config or linter updates
5. Open your PR against the QuickPulse `main` branch.
