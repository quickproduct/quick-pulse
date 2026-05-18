from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class PlanResponse(BaseModel):
    id: UUID
    name: str
    display_name: str
    max_hosts: int
    max_users: int
    max_containers: int
    metrics_retention_days: int
    price_monthly_cents: int
    price_yearly_cents: int
    features: dict | None
    is_active: bool

    model_config = {"from_attributes": True}


class SubscriptionResponse(BaseModel):
    id: UUID
    org_id: UUID
    plan_id: UUID
    status: str
    trial_ends_at: datetime | None
    current_period_start: datetime
    current_period_end: datetime
    stripe_customer_id: str | None
    cancelled_at: datetime | None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class SubscribeRequest(BaseModel):
    plan_name: str


class CancelRequest(BaseModel):
    reason: str | None = None


class UsageItem(BaseModel):
    metric_type: str
    value: int
    recorded_at: datetime


class UsageResponse(BaseModel):
    org_id: UUID
    period_start: datetime
    period_end: datetime
    usage: list[UsageItem]


class ApiKeyCreate(BaseModel):
    name: str
    scopes: list[str] = ["read"]
    expires_in_days: int | None = None


class ApiKeyResponse(BaseModel):
    id: UUID
    name: str
    key_prefix: str
    scopes: list[str] | None
    is_active: bool
    last_used_at: datetime | None
    expires_at: datetime | None
    created_at: datetime

    model_config = {"from_attributes": True}


class ApiKeyCreatedResponse(ApiKeyResponse):
    """Returned only on creation — includes the full key (not stored after this)."""
    raw_key: str
