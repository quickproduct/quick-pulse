from typing import Any

import aiodocker

from app.core.config import get_settings
from app.core.logging import get_logger

logger = get_logger("docker.client")

_docker: aiodocker.Docker | None = None


async def get_docker_client() -> aiodocker.Docker:
    global _docker
    if _docker is None:
        settings = get_settings()
        try:
            _docker = aiodocker.Docker(url=f"unix://{settings.DOCKER_SOCKET_PATH}")
            logger.info("docker_client_created", socket=settings.DOCKER_SOCKET_PATH)
        except Exception as e:
            logger.critical("docker_client_init_failed", error=str(e), socket=settings.DOCKER_SOCKET_PATH, exc_info=True)
            raise
    return _docker


async def close_docker_client() -> None:
    global _docker
    if _docker is not None:
        try:
            await _docker.close()
            logger.info("docker_client_closed")
        except Exception as e:
            logger.error("docker_client_close_failed", error=str(e), exc_info=True)
        finally:
            _docker = None


async def reset_docker_client() -> None:
    """Force-close and clear the cached client so next call re-creates it."""
    global _docker
    if _docker is not None:
        try:
            await _docker.close()
        except Exception:
            pass
        _docker = None
