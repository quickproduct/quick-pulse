"""initial schema

Revision ID: 001
Revises:
Create Date: 2024-01-01 00:00:00.000000
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

revision: str = "001"
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    op.execute("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
    op.execute("CREATE EXTENSION IF NOT EXISTS \"timescaledb\"")

    op.create_table(
        "users",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("email", sa.String(255), unique=True, nullable=False),
        sa.Column("hashed_password", sa.String(255), nullable=False),
        sa.Column("role", sa.String(20), nullable=False, server_default="admin"),
        sa.Column("is_active", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_users_email", "users", ["email"])

    op.create_table(
        "sessions",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id", ondelete="CASCADE"), nullable=False),
        sa.Column("refresh_token", sa.String(512), unique=True, nullable=False),
        sa.Column("expires_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_sessions_refresh_token", "sessions", ["refresh_token"])

    op.create_table(
        "hosts",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("hostname", sa.String(255), unique=True, nullable=False),
        sa.Column("ip_address", sa.String(45), nullable=False),
        sa.Column("os_info", sa.Text(), nullable=True),
        sa.Column("cpu_count", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("total_memory", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("total_disk", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )

    op.create_table(
        "host_metrics",
        sa.Column("time", sa.DateTime(timezone=True), primary_key=True),
        sa.Column("host_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("hosts.id", ondelete="CASCADE"), nullable=False),
        sa.Column("cpu_percent", sa.Float(), nullable=False, server_default="0"),
        sa.Column("memory_percent", sa.Float(), nullable=False, server_default="0"),
        sa.Column("memory_used", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("memory_total", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("disk_percent", sa.Float(), nullable=False, server_default="0"),
        sa.Column("disk_read_bytes", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("disk_write_bytes", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("net_bytes_sent", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("net_bytes_recv", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("load_1m", sa.Float(), nullable=False, server_default="0"),
        sa.Column("load_5m", sa.Float(), nullable=False, server_default="0"),
        sa.Column("load_15m", sa.Float(), nullable=False, server_default="0"),
        sa.Column("process_count", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("uptime_seconds", sa.Integer(), nullable=False, server_default="0"),
    )
    op.execute("SELECT create_hypertable('host_metrics', 'time', if_not_exists => TRUE)")

    op.create_table(
        "containers",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("docker_id", sa.String(64), unique=True, nullable=False),
        sa.Column("name", sa.String(255), nullable=False),
        sa.Column("image", sa.String(255), nullable=False),
        sa.Column("status", sa.String(20), nullable=False, server_default="unknown"),
        sa.Column("ports", postgresql.JSON(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_containers_docker_id", "containers", ["docker_id"])

    op.create_table(
        "container_events",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("container_docker_id", sa.String(64), nullable=True),
        sa.Column("container_name", sa.String(255), nullable=True),
        sa.Column("event_type", sa.String(50), nullable=False),
        sa.Column("timestamp", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("metadata", postgresql.JSON(), nullable=True),
    )
    op.create_index("ix_container_events_timestamp", "container_events", ["timestamp"])
    op.create_index("ix_container_events_docker_id", "container_events", ["container_docker_id"])

    op.create_table(
        "container_metrics",
        sa.Column("time", sa.DateTime(timezone=True), primary_key=True),
        sa.Column("container_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("containers.id", ondelete="CASCADE"), nullable=False),
        sa.Column("cpu_percent", sa.Float(), nullable=False, server_default="0"),
        sa.Column("memory_usage", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("memory_limit", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("network_rx", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("network_tx", sa.Integer(), nullable=False, server_default="0"),
    )
    op.execute("SELECT create_hypertable('container_metrics', 'time', if_not_exists => TRUE)")

    op.create_table(
        "compose_stacks",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("name", sa.String(255), unique=True, nullable=False),
        sa.Column("project_dir", sa.Text(), nullable=True),
        sa.Column("status", sa.String(20), nullable=False, server_default="unknown"),
        sa.Column("services_count", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )

    op.create_table(
        "compose_services",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("stack_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("compose_stacks.id", ondelete="CASCADE"), nullable=False),
        sa.Column("name", sa.String(255), nullable=False),
        sa.Column("container_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("containers.id", ondelete="SET NULL"), nullable=True),
        sa.Column("status", sa.String(20), nullable=False, server_default="unknown"),
        sa.Column("ports", postgresql.JSON(), nullable=True),
        sa.UniqueConstraint("stack_id", "name", name="uq_stack_service_name"),
    )

    op.create_table(
        "alert_rules",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("metric_type", sa.String(20), nullable=False),
        sa.Column("threshold", sa.Float(), nullable=False),
        sa.Column("operator", sa.String(10), nullable=False, server_default="gte"),
        sa.Column("duration_seconds", sa.Integer(), nullable=False, server_default="60"),
        sa.Column("enabled", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )

    op.create_table(
        "alerts",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("rule_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("alert_rules.id", ondelete="SET NULL"), nullable=True),
        sa.Column("severity", sa.String(20), nullable=False, server_default="warning"),
        sa.Column("message", sa.Text(), nullable=False),
        sa.Column("acknowledged", sa.Boolean(), nullable=False, server_default="false"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_alerts_unacked", "alerts", ["acknowledged", "created_at"])

    op.create_table(
        "audit_logs",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id", ondelete="SET NULL"), nullable=True),
        sa.Column("action", sa.String(100), nullable=False),
        sa.Column("resource_type", sa.String(100), nullable=False),
        sa.Column("resource_id", sa.String(255), nullable=True),
        sa.Column("details", postgresql.JSON(), nullable=True),
        sa.Column("timestamp", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_audit_logs_timestamp", "audit_logs", ["timestamp"])


def downgrade() -> None:
    op.drop_table("audit_logs")
    op.drop_table("alerts")
    op.drop_table("alert_rules")
    op.drop_table("compose_services")
    op.drop_table("compose_stacks")
    op.drop_table("container_metrics")
    op.drop_table("container_events")
    op.drop_table("containers")
    op.drop_table("host_metrics")
    op.drop_table("hosts")
    op.drop_table("sessions")
    op.drop_table("users")
