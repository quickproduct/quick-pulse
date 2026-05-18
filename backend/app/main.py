import asyncio
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api.v1.router import router as v1_router
from app.api.ws.metrics import router as ws_metrics_router
from app.api.ws.logs import router as ws_logs_router
from app.api.ws.events import router as ws_events_router
from app.api.ws.container_status import router as ws_container_status_router
from app.api.ws.kubernetes import router as ws_k8s_router
from app.core.config import get_settings
from app.core.logging import setup_logging
from app.infrastructure.cache.redis import close_redis, get_redis
from app.infrastructure.docker.client import close_docker_client
from app.infrastructure.websocket.manager import ws_manager


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    setup_logging(debug=settings.DEBUG, log_dir=settings.LOG_DIR)

    from app.core.logging import get_logger
    logger = get_logger("app.lifespan")
    logger.info("app_starting", version=settings.APP_VERSION, debug=settings.DEBUG, log_dir=settings.LOG_DIR)

    if settings.JWT_SECRET_KEY in ("change-me-in-production", "changeme", "secret"):
        logger.critical("insecure_jwt_secret", detail="JWT_SECRET_KEY is using a default/insecure value — change it before production use")

    # Redis — graceful degradation if unavailable
    try:
        redis = await get_redis()
        ws_manager.set_redis(redis)
        logger.info("redis_connected", url=settings.REDIS_URL)
    except Exception as e:
        logger.critical("redis_unavailable_at_startup", error=str(e), detail="Running without Redis — WebSocket pub/sub disabled")
        ws_manager.set_redis(None)

    # Docker client — workers will retry if unavailable
    try:
        import aiodocker
        docker = aiodocker.Docker(url=f"unix://{settings.DOCKER_SOCKET_PATH}")
        logger.info("docker_client_initialized", socket=settings.DOCKER_SOCKET_PATH)
    except Exception as e:
        logger.critical("docker_client_unavailable_at_startup", error=str(e), socket=settings.DOCKER_SOCKET_PATH)

    # Database engine — fatal if unavailable
    try:
        from app.infrastructure.db.database import get_engine
        engine = get_engine()
        logger.info("db_engine_initialized", pool_size=settings.DATABASE_POOL_SIZE)
    except Exception as e:
        logger.critical("db_engine_init_failed", error=str(e), exc_info=True)
        raise

    from app.workers.metrics_worker import run_metrics_worker
    from app.workers.events_worker import run_events_worker
    from app.workers.pubsub_worker import run_pubsub_worker
    from app.workers.heartbeat_worker import run_heartbeat_worker

    metrics_task = asyncio.create_task(run_metrics_worker())
    events_task = asyncio.create_task(run_events_worker())
    pubsub_task = asyncio.create_task(run_pubsub_worker())
    heartbeat_task = asyncio.create_task(run_heartbeat_worker())
    logger.info("background_workers_started")

    yield

    logger.info("app_shutting_down")
    metrics_task.cancel()
    events_task.cancel()
    pubsub_task.cancel()
    heartbeat_task.cancel()
    await asyncio.gather(metrics_task, events_task, pubsub_task, heartbeat_task, return_exceptions=True)

    try:
        await close_docker_client()
    except Exception as e:
        logger.error("docker_close_error", error=str(e))

    try:
        await close_redis()
    except Exception as e:
        logger.error("redis_close_error", error=str(e))

    try:
        from app.infrastructure.db.database import get_engine
        await get_engine().dispose()
    except Exception as e:
        logger.error("db_engine_dispose_error", error=str(e))

    logger.info("app_shutdown_complete")


def create_app() -> FastAPI:
    settings = get_settings()

    app = FastAPI(
        title=settings.APP_NAME,
        version=settings.APP_VERSION,
        lifespan=lifespan,
        debug=settings.DEBUG,
    )

    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.CORS_ORIGINS,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    app.include_router(v1_router)
    app.include_router(ws_metrics_router, prefix="/ws/metrics")
    app.include_router(ws_logs_router, prefix="/ws/logs")
    app.include_router(ws_events_router, prefix="/ws/events")
    app.include_router(ws_container_status_router, prefix="/ws/container-status")
    app.include_router(ws_k8s_router, prefix="/ws/kubernetes")

    @app.get("/health")
    async def health():
        return {"status": "ok"}

    return app


app = create_app()
