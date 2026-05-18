from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

from app.core.constants import AlertSeverity


@dataclass
class AlertRule:
    id: UUID
    metric_type: str
    threshold: float
    operator: str = "gte"
    duration_seconds: int = 60
    enabled: bool = True
    created_at: datetime | None = None


@dataclass
class Alert:
    id: UUID
    rule_id: UUID | None = None
    severity: AlertSeverity = AlertSeverity.WARNING
    message: str = ""
    acknowledged: bool = False
    created_at: datetime | None = None
