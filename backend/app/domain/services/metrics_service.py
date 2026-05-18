import asyncio
from datetime import datetime, timedelta, timezone
from uuid import UUID

from app.core.config import get_settings
from app.core.constants import HISTORY_RANGES
from app.core.logging import get_logger
from app.domain.entities.host import HostMetric
from app.domain.interfaces.repositories import HostRepository, MetricsRepository

logger = get_logger("metrics_service")


class MetricsService:
    def __init__(self, metrics_repo: MetricsRepository, host_repo: HostRepository):
        self._metrics_repo = metrics_repo
        self._host_repo = host_repo

    async def get_summary(self) -> dict:
        host = await self._host_repo.get_current()
        if not host:
            return {}
        latest = await self._metrics_repo.get_latest(host.id)
        if not latest:
            return {}
        return {
            "cpu_percent": latest.cpu_percent,
            "memory_percent": latest.memory_percent,
            "memory_used": latest.memory_used,
            "memory_total": latest.memory_total,
            "disk_percent": latest.disk_percent,
            "net_bytes_sent": latest.net_bytes_sent,
            "net_bytes_recv": latest.net_bytes_recv,
            "load_1m": latest.load_1m,
            "load_5m": latest.load_5m,
            "load_15m": latest.load_15m,
            "process_count": latest.process_count,
            "uptime_seconds": latest.uptime_seconds,
        }

    async def get_history(self, metric: str, range_key: str = "1h") -> dict:
        host = await self._host_repo.get_current()
        if not host:
            return {"metric": metric, "range": range_key, "data": []}

        range_map = {
            "1h": timedelta(hours=1),
            "24h": timedelta(hours=24),
            "7d": timedelta(days=7),
        }
        delta = range_map.get(range_key, timedelta(hours=1))
        end = datetime.now(timezone.utc)
        start = end - delta

        data = await self._metrics_repo.get_history(host.id, metric, start, end)
        return {"metric": metric, "range": range_key, "data": data}

    async def store_metric(self, host_id: UUID, raw: dict) -> None:
        metric = HostMetric(
            time=datetime.now(timezone.utc),
            host_id=host_id,
            cpu_percent=raw.get("cpu_percent", 0),
            memory_percent=raw.get("memory_percent", 0),
            memory_used=raw.get("memory_used", 0),
            memory_total=raw.get("memory_total", 0),
            disk_percent=raw.get("disk_percent", 0),
            disk_read_bytes=raw.get("disk_read_bytes", 0),
            disk_write_bytes=raw.get("disk_write_bytes", 0),
            net_bytes_sent=raw.get("net_bytes_sent", 0),
            net_bytes_recv=raw.get("net_bytes_recv", 0),
            load_1m=raw.get("load_1m", 0),
            load_5m=raw.get("load_5m", 0),
            load_15m=raw.get("load_15m", 0),
            process_count=raw.get("process_count", 0),
            uptime_seconds=raw.get("uptime_seconds", 0),
        )
        await self._metrics_repo.insert(metric)
