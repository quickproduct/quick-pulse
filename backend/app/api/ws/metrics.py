import asyncio

from fastapi import APIRouter, Query, WebSocket, WebSocketDisconnect

from app.core.logging import get_logger
from app.core.security import decode_token
from app.infrastructure.websocket.manager import ws_manager

logger = get_logger("ws.metrics")
router = APIRouter()

_CHANNEL = "metrics"


@router.websocket("")
async def ws_metrics(websocket: WebSocket, token: str = Query(None)):
    payload = _validate_token(token)
    if not payload:
        await websocket.close(code=4001, reason="Unauthorized")
        return

    user_id = payload.get("sub", "unknown")
    logger.info("ws_metrics_connect", user_id=user_id)

    await ws_manager.connect(_CHANNEL, websocket)
    try:
        while True:
            try:
                await asyncio.wait_for(websocket.receive_text(), timeout=35)
            except asyncio.TimeoutError:
                pass
    except WebSocketDisconnect:
        logger.info("ws_metrics_disconnect", user_id=user_id)
    except Exception as e:
        logger.warning("ws_metrics_error", user_id=user_id, error=str(e))
    finally:
        ws_manager.disconnect(_CHANNEL, websocket)


def _validate_token(token: str | None) -> dict | None:
    if not token:
        return None
    payload = decode_token(token)
    if not payload or payload.get("type") != "access":
        return None
    return payload
