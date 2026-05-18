"""change memory disk to bigint

Revision ID: 002
Revises: 001
Create Date: 2026-05-12 18:00:00.000000
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

revision: str = "002"
down_revision: Union[str, None] = "001"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # Change hosts table columns to BIGINT
    op.alter_column('hosts', 'total_memory', type_=postgresql.BIGINT())
    op.alter_column('hosts', 'total_disk', type_=postgresql.BIGINT())
    
    # Change host_metrics table columns to BIGINT
    op.alter_column('host_metrics', 'memory_used', type_=postgresql.BIGINT())
    op.alter_column('host_metrics', 'memory_total', type_=postgresql.BIGINT())
    op.alter_column('host_metrics', 'disk_read_bytes', type_=postgresql.BIGINT())
    op.alter_column('host_metrics', 'disk_write_bytes', type_=postgresql.BIGINT())
    op.alter_column('host_metrics', 'net_bytes_sent', type_=postgresql.BIGINT())
    op.alter_column('host_metrics', 'net_bytes_recv', type_=postgresql.BIGINT())
    op.alter_column('host_metrics', 'uptime_seconds', type_=postgresql.BIGINT())


def downgrade() -> None:
    # Revert hosts table columns to INTEGER
    op.alter_column('hosts', 'total_memory', type_=sa.Integer())
    op.alter_column('hosts', 'total_disk', type_=sa.Integer())
    
    # Revert host_metrics table columns to INTEGER
    op.alter_column('host_metrics', 'memory_used', type_=sa.Integer())
    op.alter_column('host_metrics', 'memory_total', type_=sa.Integer())
    op.alter_column('host_metrics', 'disk_read_bytes', type_=sa.Integer())
    op.alter_column('host_metrics', 'disk_write_bytes', type_=sa.Integer())
    op.alter_column('host_metrics', 'net_bytes_sent', type_=sa.Integer())
    op.alter_column('host_metrics', 'net_bytes_recv', type_=sa.Integer())
    op.alter_column('host_metrics', 'uptime_seconds', type_=sa.Integer())
