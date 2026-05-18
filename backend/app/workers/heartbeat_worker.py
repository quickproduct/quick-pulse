import asyncio
from app.infrastructure.websocket.manager import ws_manager
from app.core.logging import get_logger

logger = get_logger("workers.heartbeat")

async def run_heartbeat_worker():
    """
    Background worker that sends periodic pings to all connected 
    WebSocket clients to keep connections alive and detect stale ones.
    """
    logger.info("heartbeat_worker_starting")
    try:
        await ws_manager.run_heartbeat()
    except asyncio.CancelledError:
        logger.info("heartbeat_worker_cancelled")
    except Exception as e:
        logger.error("heartbeat_worker_error", error=str(e), exc_info=True)
