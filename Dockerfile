# ── Stage 1: Build Frontend ───────────────────────────────────────────────
FROM node:20-slim AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline
COPY frontend/ ./
RUN npm run build

# ── Stage 2: Build Go Backend ─────────────────────────────────────────────
FROM golang:1.26-alpine AS backend-builder
# We need build tools to download dependencies (git, etc.) if required
RUN apk add --no-cache git gcc musl-dev
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Copy pre-built frontend into the embed location (internal/static/build)
COPY --from=frontend-builder /app/build ./internal/static/build
# Compile Go backend as a statically linked binary (CGO_ENABLED=0 since glebarez/modernc sqlite is CGO-free)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o qp-backend ./cmd/server

# ── Stage 3: Minimal Runtime ──────────────────────────────────────────────
FROM alpine:3.19 AS runtime
# Install curl for healthcheck and ca-certificates for potential external API requests
RUN apk add --no-cache curl ca-certificates tzdata

WORKDIR /app
# Copy binary from builder
COPY --from=backend-builder /app/qp-backend ./qp-backend

# Ensure data directory exists
RUN mkdir -p /app/data

EXPOSE 8000

# Set environment defaults
ENV PORT=8000 \
    SQLITE_DB_PATH=/app/data/quickpulse.db \
    GIN_MODE=release

HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

CMD ["./qp-backend"]
