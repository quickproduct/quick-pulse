from dataclasses import dataclass, field
from datetime import datetime
from uuid import UUID


@dataclass
class Plan:
    name: str
    display_name: str
    max_hosts: int = 1
    max_users: int = 2
    max_containers: int = 10
    metrics_retention_days: int = 7
    price_monthly_cents: int = 0
    price_yearly_cents: int = 0
    features: dict | None = None
    is_active: bool = True
    id: UUID | None = None
    created_at: datetime | None = None
