import asyncio
import json
from typing import Any

from app.core.constants import EventType
from app.core.logging import get_logger
from app.domain.entities.event import ContainerEvent
from app.domain.interfaces.repositories import ContainerEventRepository
from app.infrastructure.websocket.manager import WebSocketManager

logger = get_logger("event_stream_service")

_DOCKER_EVENT_MAP = {
    "start": EventType.CONTAINER_START,
    "stop": EventType.CONTAINER_STOP,
    "restart": EventType.CONTAINER_RESTART,
    "die": EventType.CONTAINER_DIE,
    "health_status": EventType.CONTAINER_HEALTH,
    "create": EventType.CONTAINER_CREATE,
    "destroy": EventType.CONTAINER_DESTROY,
}


class EventStreamService:
    def __init__(self, docker: Any, event_repo: ContainerEventRepository, ws_manager: WebSocketManager):
        self._docker = docker
        self._event_repo = event_repo
        self._ws_manager = ws_manager

    async def consume_events(self) -> None:
        logger.info("event_consumer_started")
        try:
            subscriber = self._docker.events.subscribe()
            while True:
                event = await subscriber.get()

                # subscriber.get() returns None when the stream is exhausted
                if event is None:
                    logger.warning("event_subscriber_returned_none", detail="Docker event stream ended; will reconnect")
                    break

                try:
                    event_data = json.loads(event) if isinstance(event, (bytes, str)) else event
                except (json.JSONDecodeError, TypeError) as e:
                    logger.warning("event_json_parse_error", error=str(e))
                    continue

                if not isinstance(event_data, dict):
                    logger.warning("event_unexpected_type", type=type(event_data).__name__)
                    continue

                action = event_data.get("Action", "")
                event_type = event_data.get("Type", "")

                if event_type != "container":
                    continue

                mapped_type = _DOCKER_EVENT_MAP.get(action)
                if not mapped_type:
                    continue

                actor = event_data.get("Actor") or {}
                attrs = actor.get("Attributes") or {}
                project = attrs.get("com.docker.compose.project", "")
                container_id = (actor.get("ID") or "")[:12]
                container_name = attrs.get("name", container_id)

                # Filter out QuickPulse's own infrastructure events
                if container_name.startswith("qp-") or project == "quickpulse":
                    continue

                try:
                    container_event = ContainerEvent(
                        container_docker_id=container_id,
                        container_name=container_name,
                        event_type=mapped_type,
                        metadata=attrs,
                    )
                    saved = await self._event_repo.insert(container_event)
                except Exception as e:
                    logger.error("event_repo_insert_failed", action=action, container=container_name, error=str(e), exc_info=True)
                    continue

                try:
                    await self._ws_manager.broadcast("events", {
                        "id": str(saved.id),
                        "container_docker_id": container_id,
                        "container_name": container_name,
                        "event_type": mapped_type.value,
                        "timestamp": saved.timestamp.isoformat() if saved.timestamp else None,
                        "metadata": attrs,
                    })
                except Exception as e:
                    logger.warning("event_ws_broadcast_failed", channel="events", error=str(e))

                try:
                    await self._ws_manager.broadcast("container-status", {
                        "container_id": container_id,
                        "name": container_name,
                        "status": action,
                        "timestamp": saved.timestamp.isoformat() if saved.timestamp else None,
                    })
                except Exception as e:
                    logger.warning("event_ws_broadcast_failed", channel="container-status", error=str(e))

                logger.debug("docker_event", action=action, container=container_name)

        except asyncio.CancelledError:
            logger.info("event_consumer_stopped")
            raise
        except Exception as e:
            logger.error("event_consumer_error", error=str(e), exc_info=True)
            raise
