from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class EventResponse(BaseModel):
    id: UUID | None = None
    container_docker_id: str | None = None
    container_name: str | None = None
    event_type: str
    timestamp: datetime | None = None
    metadata: dict | None = None
