from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

from app.core.constants import ContainerStatus


@dataclass
class Container:
    id: UUID
    docker_id: str
    name: str
    image: str
    status: ContainerStatus = ContainerStatus.UNKNOWN
    ports: dict | None = None
    created_at: datetime | None = None
    updated_at: datetime | None = None
