from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

from app.core.constants import ContainerStatus, StackStatus


@dataclass
class ComposeStack:
    id: UUID
    name: str
    project_dir: str | None = None
    status: StackStatus = StackStatus.UNKNOWN
    services_count: int = 0
    created_at: datetime | None = None
    updated_at: datetime | None = None


@dataclass
class ComposeService:
    id: UUID
    stack_id: UUID
    name: str
    container_id: UUID | None = None
    status: ContainerStatus = ContainerStatus.UNKNOWN
    ports: dict | None = None
