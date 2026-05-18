from datetime import datetime
from uuid import UUID

from sqlalchemy import text, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.domain.entities.host import HostMetric
from app.domain.interfaces.repositories import MetricsRepository
from app.infrastructure.db.models import HostMetricModel


class SqlMetricsRepository(MetricsRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def insert(self, metric: HostMetric) -> None:
        model = HostMetricModel(
            host_id=metric.host_id,
            cpu_percent=metric.cpu_percent,
            memory_percent=metric.memory_percent,
            memory_used=metric.memory_used,
            memory_total=metric.memory_total,
            disk_percent=metric.disk_percent,
            disk_read_bytes=metric.disk_read_bytes,
            disk_write_bytes=metric.disk_write_bytes,
            net_bytes_sent=metric.net_bytes_sent,
            net_bytes_recv=metric.net_bytes_recv,
            load_1m=metric.load_1m,
            load_5m=metric.load_5m,
            load_15m=metric.load_15m,
            process_count=metric.process_count,
            uptime_seconds=metric.uptime_seconds,
        )
        self._session.add(model)
        await self._session.flush()

    async def get_latest(self, host_id: UUID) -> HostMetric | None:
        result = await self._session.execute(
            select(HostMetricModel)
            .where(HostMetricModel.host_id == host_id)
            .order_by(HostMetricModel.time.desc())
            .limit(1)
        )
        model = result.scalar_one_or_none()
        if not model:
            return None
        return HostMetric(
            time=model.time,
            host_id=model.host_id,
            cpu_percent=model.cpu_percent,
            memory_percent=model.memory_percent,
            memory_used=model.memory_used,
            memory_total=model.memory_total,
            disk_percent=model.disk_percent,
            disk_read_bytes=model.disk_read_bytes,
            disk_write_bytes=model.disk_write_bytes,
            net_bytes_sent=model.net_bytes_sent,
            net_bytes_recv=model.net_bytes_recv,
            load_1m=model.load_1m,
            load_5m=model.load_5m,
            load_15m=model.load_15m,
            process_count=model.process_count,
            uptime_seconds=model.uptime_seconds,
        )

    async def get_history(self, host_id: UUID, metric: str, start: datetime, end: datetime) -> list[dict]:
        metric_col_map = {
            "cpu": "cpu_percent",
            "memory": "memory_percent",
            "disk": "disk_percent",
            "network": "net_bytes_recv",
            "load": "load_1m",
        }
        col = metric_col_map.get(metric, "cpu_percent")
        # Guard against unexpected col values (metric_col_map is the only source, but be explicit)
        assert col in {"cpu_percent", "memory_percent", "disk_percent", "net_bytes_recv", "load_1m"}, f"Unexpected column: {col}"

        query = text(f"""
            SELECT time_bucket('1 minute', time) AS bucket, avg({col}) AS value
            FROM host_metrics
            WHERE host_id = :host_id AND time >= :start AND time <= :end
            GROUP BY bucket ORDER BY bucket
        """)
        result = await self._session.execute(query, {"host_id": str(host_id), "start": start, "end": end})
        return [{"time": row[0].isoformat() if row[0] else None, "value": round(float(row[1] or 0), 2)} for row in result.fetchall()]
