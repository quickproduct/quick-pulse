# ── Stage 1: dependency builder ────────────────────────────────────────────
FROM python:3.12-slim AS deps

WORKDIR /build

RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc libpq-dev \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

# ── Stage 2: runtime (minimal footprint) ───────────────────────────────────
FROM python:3.12-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends libpq5 curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1001 appuser

WORKDIR /app

# Copy pre-built deps from builder stage
COPY --from=deps /root/.local /home/appuser/.local

# Copy application source (owned by appuser)
COPY --chown=appuser:appuser . .

# Ensure logs directory is owned by appuser before volume mount
RUN mkdir -p /app/logs && chown appuser:appuser /app/logs

USER appuser

ENV PATH=/home/appuser/.local/bin:$PATH \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONPATH=/app

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
