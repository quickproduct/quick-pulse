from dataclasses import dataclass
from datetime import datetime
from uuid import UUID


@dataclass
class Subscription:
    org_id: UUID
    plan_id: UUID
    status: str
    current_period_start: datetime
    current_period_end: datetime
    id: UUID | None = None
    trial_ends_at: datetime | None = None
    stripe_customer_id: str | None = None
    stripe_subscription_id: str | None = None
    cancelled_at: datetime | None = None
    created_at: datetime | None = None
    updated_at: datetime | None = None


@dataclass
class ApiKey:
    org_id: UUID
    user_id: UUID
    name: str
    key_hash: str
    key_prefix: str
    scopes: list[str] | None = None
    is_active: bool = True
    id: UUID | None = None
    last_used_at: datetime | None = None
    expires_at: datetime | None = None
    created_at: datetime | None = None


@dataclass
class UsageRecord:
    org_id: UUID
    metric_type: str
    value: int
    id: UUID | None = None
    recorded_at: datetime | None = None
