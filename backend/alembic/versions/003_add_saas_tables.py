"""add saas tables: plans, organizations, billing, api keys

Revision ID: 003
Revises: 002
Create Date: 2026-05-13 00:00:00.000000
"""
import uuid
from datetime import datetime, timezone, timedelta
from typing import Sequence, Union

import sqlalchemy as sa
from alembic import op
from sqlalchemy.dialects import postgresql

revision: str = "003"
down_revision: Union[str, None] = "002"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # ── plans ─────────────────────────────────────────────────────────────────
    op.create_table(
        "plans",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("name", sa.String(50), unique=True, nullable=False),
        sa.Column("display_name", sa.String(100), nullable=False),
        sa.Column("max_hosts", sa.Integer(), nullable=False, server_default="1"),
        sa.Column("max_users", sa.Integer(), nullable=False, server_default="2"),
        sa.Column("max_containers", sa.Integer(), nullable=False, server_default="10"),
        sa.Column("metrics_retention_days", sa.Integer(), nullable=False, server_default="7"),
        sa.Column("price_monthly_cents", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("price_yearly_cents", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("features", postgresql.JSON(), nullable=True),
        sa.Column("is_active", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )

    # ── organizations ─────────────────────────────────────────────────────────
    op.create_table(
        "organizations",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("name", sa.String(255), nullable=False),
        sa.Column("slug", sa.String(100), unique=True, nullable=False),
        sa.Column("plan_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("plans.id", ondelete="SET NULL"), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_organizations_slug", "organizations", ["slug"])

    # ── org_members ───────────────────────────────────────────────────────────
    op.create_table(
        "org_members",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="CASCADE"), nullable=False),
        sa.Column("user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id", ondelete="CASCADE"), nullable=False),
        sa.Column("role", sa.String(20), nullable=False, server_default="member"),
        sa.Column("invited_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id", ondelete="SET NULL"), nullable=True),
        sa.Column("joined_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.UniqueConstraint("org_id", "user_id", name="uq_org_member"),
    )

    # ── org_invitations ───────────────────────────────────────────────────────
    op.create_table(
        "org_invitations",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="CASCADE"), nullable=False),
        sa.Column("email", sa.String(255), nullable=False),
        sa.Column("role", sa.String(20), nullable=False, server_default="member"),
        sa.Column("token", sa.String(64), unique=True, nullable=False),
        sa.Column("invited_by", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id", ondelete="CASCADE"), nullable=False),
        sa.Column("expires_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("accepted_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_org_invitations_token", "org_invitations", ["token"])

    # ── subscriptions ─────────────────────────────────────────────────────────
    op.create_table(
        "subscriptions",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="CASCADE"), unique=True, nullable=False),
        sa.Column("plan_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("plans.id", ondelete="RESTRICT"), nullable=False),
        sa.Column("status", sa.String(20), nullable=False, server_default="trial"),
        sa.Column("trial_ends_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("current_period_start", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("current_period_end", sa.DateTime(timezone=True), nullable=False),
        sa.Column("stripe_customer_id", sa.String(100), nullable=True),
        sa.Column("stripe_subscription_id", sa.String(100), nullable=True),
        sa.Column("cancelled_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )

    # ── usage_records ─────────────────────────────────────────────────────────
    op.create_table(
        "usage_records",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="CASCADE"), nullable=False),
        sa.Column("metric_type", sa.String(50), nullable=False),
        sa.Column("value", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("recorded_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_usage_records_recorded_at", "usage_records", ["recorded_at"])
    op.create_index("ix_usage_org_metric_time", "usage_records", ["org_id", "metric_type", "recorded_at"])

    # ── api_keys ──────────────────────────────────────────────────────────────
    op.create_table(
        "api_keys",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="CASCADE"), nullable=False),
        sa.Column("user_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("users.id", ondelete="CASCADE"), nullable=False),
        sa.Column("name", sa.String(100), nullable=False),
        sa.Column("key_hash", sa.String(64), unique=True, nullable=False),
        sa.Column("key_prefix", sa.String(12), nullable=False),
        sa.Column("scopes", postgresql.JSON(), nullable=True),
        sa.Column("last_used_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("expires_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("is_active", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_api_keys_key_hash", "api_keys", ["key_hash"])

    # ── org_id on existing tables (nullable for backwards compat) ─────────────
    op.add_column("hosts", sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="SET NULL"), nullable=True))
    op.add_column("containers", sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="SET NULL"), nullable=True))
    op.add_column("compose_stacks", sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="SET NULL"), nullable=True))
    op.add_column("alert_rules", sa.Column("org_id", postgresql.UUID(as_uuid=True), sa.ForeignKey("organizations.id", ondelete="SET NULL"), nullable=True))

    # ── seed default plans ────────────────────────────────────────────────────
    now = datetime.now(timezone.utc)
    op.bulk_insert(
        sa.table(
            "plans",
            sa.column("id", postgresql.UUID(as_uuid=True)),
            sa.column("name", sa.String),
            sa.column("display_name", sa.String),
            sa.column("max_hosts", sa.Integer),
            sa.column("max_users", sa.Integer),
            sa.column("max_containers", sa.Integer),
            sa.column("metrics_retention_days", sa.Integer),
            sa.column("price_monthly_cents", sa.Integer),
            sa.column("price_yearly_cents", sa.Integer),
            sa.column("features", postgresql.JSON),
            sa.column("is_active", sa.Boolean),
            sa.column("created_at", sa.DateTime(timezone=True)),
        ),
        [
            {
                "id": str(uuid.uuid4()),
                "name": "free",
                "display_name": "Free",
                "max_hosts": 1,
                "max_users": 2,
                "max_containers": 10,
                "metrics_retention_days": 7,
                "price_monthly_cents": 0,
                "price_yearly_cents": 0,
                "features": {"alerts": True, "api_keys": False, "multi_host": False},
                "is_active": True,
                "created_at": now,
            },
            {
                "id": str(uuid.uuid4()),
                "name": "pro",
                "display_name": "Pro",
                "max_hosts": 5,
                "max_users": 10,
                "max_containers": -1,
                "metrics_retention_days": 30,
                "price_monthly_cents": 1900,
                "price_yearly_cents": 19900,
                "features": {"alerts": True, "api_keys": True, "multi_host": True},
                "is_active": True,
                "created_at": now,
            },
            {
                "id": str(uuid.uuid4()),
                "name": "enterprise",
                "display_name": "Enterprise",
                "max_hosts": -1,
                "max_users": -1,
                "max_containers": -1,
                "metrics_retention_days": 365,
                "price_monthly_cents": 9900,
                "price_yearly_cents": 99900,
                "features": {"alerts": True, "api_keys": True, "multi_host": True, "sso": True, "priority_support": True},
                "is_active": True,
                "created_at": now,
            },
        ],
    )


def downgrade() -> None:
    op.drop_column("alert_rules", "org_id")
    op.drop_column("compose_stacks", "org_id")
    op.drop_column("containers", "org_id")
    op.drop_column("hosts", "org_id")

    op.drop_index("ix_api_keys_key_hash", table_name="api_keys")
    op.drop_table("api_keys")

    op.drop_index("ix_usage_org_metric_time", table_name="usage_records")
    op.drop_index("ix_usage_records_recorded_at", table_name="usage_records")
    op.drop_table("usage_records")

    op.drop_table("subscriptions")

    op.drop_index("ix_org_invitations_token", table_name="org_invitations")
    op.drop_table("org_invitations")

    op.drop_table("org_members")

    op.drop_index("ix_organizations_slug", table_name="organizations")
    op.drop_table("organizations")

    op.drop_table("plans")
