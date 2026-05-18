import redis.asyncio as aioredis

from app.core.config import get_settings
from app.core.logging import get_logger

logger = get_logger("cache.redis")

_redis: aioredis.Redis | None = None


async def get_redis() -> aioredis.Redis:
    global _redis
    if _redis is None:
        settings = get_settings()
        try:
            _redis = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
            await _redis.ping()
            logger.info("redis_connected", url=settings.REDIS_URL)
        except Exception as e:
            logger.critical("redis_connect_failed", error=str(e), url=settings.REDIS_URL, exc_info=True)
            _redis = None
            raise
    return _redis


async def close_redis() -> None:
    global _redis
    if _redis is not None:
        try:
            await _redis.aclose()
            logger.info("redis_closed")
        except Exception as e:
            logger.error("redis_close_failed", error=str(e), exc_info=True)
        finally:
            _redis = None
