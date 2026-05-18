import asyncio
from typing import Any

from app.core.constants import EventType
from app.core.logging import get_logger
from app.domain.entities.event import ContainerEvent
from app.domain.interfaces.repositories import ContainerEventRepository
from app.infrastructure.websocket.manager import WebSocketManager

logger = get_logger("logs_service")

_DOCKER_EVENT_MAP = {
    "start": EventType.CONTAINER_START,
    "stop": EventType.CONTAINER_STOP,
    "restart": EventType.CONTAINER_RESTART,
    "die": EventType.CONTAINER_DIE,
    "health_status": EventType.CONTAINER_HEALTH,
    "create": EventType.CONTAINER_CREATE,
    "destroy": EventType.CONTAINER_DESTROY,
}


class LogsService:
    def __init__(self, docker: Any, event_repo: ContainerEventRepository, ws_manager: WebSocketManager):
        self._docker = docker
        self._event_repo = event_repo
        self._ws_manager = ws_manager

    async def stream_logs(self, container_id: str, tail: int = 100):
        container = self._docker.containers.container(container_id)
        stream = container.log(stdout=True, stderr=True, follow=True, tail=tail)
        async for line in stream:
            yield line

    async def get_logs(self, container_id: str, tail: int = 100, since: str | None = None) -> list[str]:
        container = self._docker.containers.container(container_id)
        kwargs = {"stdout": True, "stderr": True, "tail": tail}
        if since:
            kwargs["since"] = since
        logs = await container.log(**kwargs)
        return logs if isinstance(logs, list) else [logs]

    async def process_docker_event(self, event: dict) -> ContainerEvent | None:
        action = event.get("Action", "")
        event_type_str = event.get("Type", "")
        if event_type_str != "container":
            return None

        mapped_type = _DOCKER_EVENT_MAP.get(action)
        if not mapped_type:
            return None

        actor = event.get("Actor", {})
        attrs = actor.get("Attributes", {})
        container_id = actor.get("ID", "")[:12]
        container_name = attrs.get("name", container_id)

        container_event = ContainerEvent(
            container_docker_id=container_id,
            container_name=container_name,
            event_type=mapped_type,
            metadata=attrs,
        )
        saved = await self._event_repo.insert(container_event)

        await self._ws_manager.broadcast("events", {
            "id": str(saved.id),
            "container_docker_id": container_id,
            "container_name": container_name,
            "event_type": mapped_type.value,
            "timestamp": saved.timestamp.isoformat() if saved.timestamp else None,
            "metadata": attrs,
        })

        await self._ws_manager.broadcast("container-status", {
            "container_id": container_id,
            "name": container_name,
            "status": action,
            "timestamp": saved.timestamp.isoformat() if saved.timestamp else None,
        })

        return saved
