from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class StackResponse(BaseModel):
    id: UUID
    name: str
    project_dir: str | None = None
    status: str
    services_count: int
    created_at: datetime | None = None


class StackDetailResponse(StackResponse):
    services: list[dict] = []


class StackActionResponse(BaseModel):
    success: bool
    message: str
    stack_name: str
