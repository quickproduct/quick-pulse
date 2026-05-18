from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class AlertRuleCreate(BaseModel):
    metric_type: str
    threshold: float
    operator: str = "gte"
    duration_seconds: int = 60
    enabled: bool = True


class AlertRuleUpdate(BaseModel):
    threshold: float | None = None
    operator: str | None = None
    duration_seconds: int | None = None
    enabled: bool | None = None


class AlertRuleResponse(BaseModel):
    id: UUID
    metric_type: str
    threshold: float
    operator: str
    duration_seconds: int
    enabled: bool
    created_at: datetime | None = None


class AlertResponse(BaseModel):
    id: UUID
    rule_id: UUID | None = None
    severity: str
    message: str
    acknowledged: bool
    created_at: datetime | None = None
