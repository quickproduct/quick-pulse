import asyncio

from app.core.logging import get_logger
from app.infrastructure.db.database import get_session_factory
from app.infrastructure.docker.client import get_docker_client, reset_docker_client
from app.repositories.container_event_repo import SqlContainerEventRepository
from app.domain.services.event_stream_service import EventStreamService
from app.infrastructure.websocket.manager import ws_manager

logger = get_logger("workers.events")

_MAX_CONSECUTIVE_ERRORS = 5


async def run_events_worker() -> None:
    logger.info("events_worker_starting")
    consecutive_errors = 0

    while True:
        try:
            docker = await get_docker_client()
            # Fresh session per reconnect — never reuse across connection attempts
            session_factory = get_session_factory()
            logger.info("events_worker_listening")
            async with session_factory() as session:
                event_repo = SqlContainerEventRepository(session)
                service = EventStreamService(docker, event_repo, ws_manager)
                await service.consume_events()
            # consume_events returned normally (stream ended) — reconnect immediately
            consecutive_errors = 0
        except asyncio.CancelledError:
            logger.info("events_worker_stopped")
            return
        except Exception as e:
            consecutive_errors += 1
            logger.error("events_worker_error", error=str(e), consecutive=consecutive_errors, exc_info=True)

            if consecutive_errors >= _MAX_CONSECUTIVE_ERRORS:
                logger.critical(
                    "events_worker_circuit_open",
                    consecutive=consecutive_errors,
                    detail="Too many consecutive failures; Docker may be unavailable",
                )

            # Reset the Docker client so next iteration gets a fresh connection
            try:
                await reset_docker_client()
            except Exception:
                pass

            delay = min(5 * consecutive_errors, 60)
            logger.info("events_worker_reconnecting", delay_seconds=delay)
            await asyncio.sleep(delay)
