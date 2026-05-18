import asyncio

from fastapi import APIRouter, Query, WebSocket, WebSocketDisconnect

from app.core.logging import get_logger
from app.core.security import decode_token
from app.infrastructure.websocket.manager import ws_manager

logger = get_logger("ws.logs")
router = APIRouter()


@router.websocket("/{container_id}")
async def ws_logs(websocket: WebSocket, container_id: str, token: str = Query(None)):
    payload = _validate_token(token)
    if not payload:
        await websocket.close(code=4001, reason="Unauthorized")
        return

    user_id = payload.get("sub", "unknown")
    channel = f"logs:{container_id}"
    logger.info("ws_logs_connect", container_id=container_id, user_id=user_id)

    await ws_manager.connect(channel, websocket)
    paused = False

    try:
        from app.infrastructure.docker.client import get_docker_client
        docker = await get_docker_client()
        container = docker.containers.container(container_id)
        log_stream = container.log(stdout=True, stderr=True, follow=True, tail=100)

        async def read_logs():
            nonlocal paused
            try:
                async for line in log_stream:
                    if not paused:
                        try:
                            await websocket.send_json({"line": line, "container_id": container_id})
                        except Exception as e:
                            logger.warning("ws_logs_send_failed", container_id=container_id, error=str(e))
                            break
            except Exception as e:
                logger.warning("ws_logs_stream_error", container_id=container_id, error=str(e))

        async def read_commands():
            nonlocal paused
            while True:
                try:
                    data = await websocket.receive_json()
                    if data.get("action") == "pause":
                        paused = True
                    elif data.get("action") == "resume":
                        paused = False
                except WebSocketDisconnect:
                    break
                except Exception as e:
                    logger.warning("ws_logs_command_error", container_id=container_id, error=str(e))
                    break

        results = await asyncio.gather(read_logs(), read_commands(), return_exceptions=True)
        for r in results:
            if isinstance(r, Exception) and not isinstance(r, asyncio.CancelledError):
                logger.warning("ws_logs_task_exception", container_id=container_id, error=str(r))

    except WebSocketDisconnect:
        logger.info("ws_logs_disconnect", container_id=container_id, user_id=user_id)
    except Exception as e:
        logger.error("ws_logs_error", container_id=container_id, user_id=user_id, error=str(e), exc_info=True)
    finally:
        ws_manager.disconnect(channel, websocket)
        logger.info("ws_logs_cleaned_up", container_id=container_id)


def _validate_token(token: str | None) -> dict | None:
    if not token:
        return None
    payload = decode_token(token)
    if not payload or payload.get("type") != "access":
        return None
    return payload
