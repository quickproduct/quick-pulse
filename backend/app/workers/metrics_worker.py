import asyncio
import socket
import platform
from datetime import datetime, timezone

import psutil

from app.core.config import get_settings
from app.core.logging import get_logger
from app.domain.entities.host import Host
from app.infrastructure.db.database import get_session_factory
from app.infrastructure.metrics.collector import collect_host_metrics
from app.infrastructure.websocket.manager import ws_manager
from app.repositories.host_repo import SqlHostRepository
from app.repositories.metrics_repo import SqlMetricsRepository
from app.domain.services.metrics_service import MetricsService
from app.domain.services.alert_service import AlertService
from app.repositories.alert_repo import SqlAlertRuleRepository, SqlAlertRepository

logger = get_logger("workers.metrics")

_MAX_CONSECUTIVE_ERRORS = 5
_BACKOFF_BASE_SECONDS = 5
_BACKOFF_MAX_SECONDS = 300  # 5 minutes cap


async def run_metrics_worker() -> None:
    settings = get_settings()
    interval = settings.METRICS_COLLECTION_INTERVAL_SECONDS
    logger.info("metrics_worker_starting", interval_seconds=interval)
    session_factory = get_session_factory()
    consecutive_errors = 0

    while True:
        try:
            raw = collect_host_metrics()
            if not raw:
                logger.warning("metrics_collection_empty")
                await asyncio.sleep(interval)
                continue

            logger.debug(
                "metrics_collected",
                cpu_percent=raw.get("cpu_percent"),
                memory_percent=raw.get("memory_percent"),
                disk_percent=raw.get("disk_percent"),
            )

            async with session_factory() as session:
                host_repo = SqlHostRepository(session)
                metrics_repo = SqlMetricsRepository(session)
                alert_rule_repo = SqlAlertRuleRepository(session)
                alert_repo = SqlAlertRepository(session)

                host = await host_repo.get_current()
                if not host:
                    hostname = socket.gethostname()
                    try:
                        ip_address = socket.gethostbyname(hostname)
                    except Exception:
                        ip_address = "127.0.0.1"

                    host = Host(
                        id=None,
                        hostname=hostname,
                        ip_address=ip_address,
                        os_info=platform.system(),
                        cpu_count=psutil.cpu_count() or 0,
                        total_memory=psutil.virtual_memory().total,
                        total_disk=psutil.disk_usage("/").total,
                    )
                    host = await host_repo.upsert(host)
                    await session.commit()
                    logger.info("host_registered", hostname=hostname, ip=ip_address)
                else:
                    logger.debug("host_found", host_id=str(host.id), hostname=host.hostname)

                metrics_service = MetricsService(metrics_repo, host_repo)
                await metrics_service.store_metric(host.id, raw)
                logger.debug("metrics_stored", host_id=str(host.id))

                try:
                    alert_service = AlertService(alert_repo, alert_rule_repo)
                    fired = await alert_service.evaluate_rules(raw)
                    if fired:
                        logger.warning("alerts_fired", count=len(fired) if hasattr(fired, "__len__") else 1)
                    else:
                        logger.debug("alert_evaluation_ok")
                except Exception as e:
                    logger.warning("alert_evaluation_error", error=str(e), exc_info=True)

                await session.commit()

            try:
                await ws_manager.broadcast("metrics", {
                    **raw,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                })
                logger.debug("metrics_broadcast_sent")
            except Exception as e:
                logger.warning("metrics_broadcast_failed", error=str(e))

            consecutive_errors = 0

        except asyncio.CancelledError:
            logger.info("metrics_worker_stopped")
            return
        except Exception as e:
            consecutive_errors += 1
            logger.error("metrics_worker_error", error=str(e), consecutive=consecutive_errors, exc_info=True)

            if consecutive_errors >= _MAX_CONSECUTIVE_ERRORS:
                # Exponential backoff: 5s * N, capped at 5 minutes
                backoff = min(_BACKOFF_BASE_SECONDS * consecutive_errors, _BACKOFF_MAX_SECONDS)
                logger.critical(
                    "metrics_worker_circuit_open",
                    consecutive=consecutive_errors,
                    backoff_seconds=backoff,
                    detail="Too many consecutive failures; backing off before retry",
                )
                await asyncio.sleep(backoff)
                continue

        await asyncio.sleep(interval)
