from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class DashboardResponse(BaseModel):
    host: dict | None = None
    metrics: dict | None = None
    containers: dict | None = None
    recent_events: list[dict] = []
    stack_summary: dict | None = None
