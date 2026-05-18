import asyncio
from app.infrastructure.websocket.manager import ws_manager
from app.core.logging import get_logger

logger = get_logger("workers.pubsub")

async def run_pubsub_worker():
    """
    Background worker that listens to Redis Pub/Sub channels
    and broadcasts messages to connected WebSocket clients.
    """
    logger.info("pubsub_worker_starting")
    try:
        await ws_manager.subscribe_to_redis_channels()
    except asyncio.CancelledError:
        logger.info("pubsub_worker_cancelled")
    except Exception as e:
        logger.error("pubsub_worker_error", error=str(e), exc_info=True)
