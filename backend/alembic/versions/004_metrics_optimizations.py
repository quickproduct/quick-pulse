"""metrics optimizations: compression and retention policies

Revision ID: 004
Revises: 003
Create Date: 2026-05-13 12:00:00.000000
"""
from typing import Sequence, Union
from alembic import op

revision: str = "004"
down_revision: Union[str, None] = "003"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # ── host_metrics ──────────────────────────────────────────────────────────
    # Enable compression for host_metrics after 7 days
    op.execute("ALTER TABLE host_metrics SET (timescaledb.compress, timescaledb.compress_segmentby = 'host_id')")
    op.execute("SELECT add_compression_policy('host_metrics', INTERVAL '7 days')")
    
    # Add retention policy for host_metrics (default 30 days)
    op.execute("SELECT add_retention_policy('host_metrics', INTERVAL '30 days')")

    # ── container_metrics ─────────────────────────────────────────────────────
    # Enable compression for container_metrics after 3 days (higher volume)
    op.execute("ALTER TABLE container_metrics SET (timescaledb.compress, timescaledb.compress_segmentby = 'container_id')")
    op.execute("SELECT add_compression_policy('container_metrics', INTERVAL '3 days')")
    
    # Add retention policy for container_metrics (14 days)
    op.execute("SELECT add_retention_policy('container_metrics', INTERVAL '14 days')")


def downgrade() -> None:
    # Remove policies
    op.execute("SELECT remove_retention_policy('container_metrics', if_exists => TRUE)")
    op.execute("SELECT remove_compression_policy('container_metrics', if_exists => TRUE)")
    
    op.execute("SELECT remove_retention_policy('host_metrics', if_exists => TRUE)")
    op.execute("SELECT remove_compression_policy('host_metrics', if_exists => TRUE)")
    
    # Disable compression
    op.execute("ALTER TABLE container_metrics SET (timescaledb.compress = FALSE)")
    op.execute("ALTER TABLE host_metrics SET (timescaledb.compress = FALSE)")
