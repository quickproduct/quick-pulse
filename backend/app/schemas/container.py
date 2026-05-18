from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class ContainerResponse(BaseModel):
    id: UUID | None = None
    docker_id: str
    name: str
    image: str
    status: str
    ports: dict | list | None = None
    created_at: datetime | None = None
    updated_at: datetime | None = None


class ContainerDetailResponse(ContainerResponse):
    env: list[str] | None = None
    network_settings: dict | None = None
    mounts: list | None = None
    resource_usage: dict | None = None


class ContainerActionResponse(BaseModel):
    success: bool
    message: str
    container_id: str


class LogRequest(BaseModel):
    tail: int = 100
    since: str | None = None
    stdout: bool = True
    stderr: bool = True


class LogLine(BaseModel):
    timestamp: str | None = None
    line: str
    stream: str = "stdout"
