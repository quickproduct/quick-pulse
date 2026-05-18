from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

from app.core.constants import EventType


@dataclass
class ContainerEvent:
    id: UUID | None = None
    container_docker_id: str | None = None
    container_name: str | None = None
    event_type: EventType = EventType.CONTAINER_START
    timestamp: datetime | None = None
    metadata: dict | None = None
