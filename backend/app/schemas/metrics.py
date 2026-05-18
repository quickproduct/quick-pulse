from datetime import datetime
from uuid import UUID

from pydantic import BaseModel


class MetricSnapshot(BaseModel):
    cpu_percent: float = 0
    memory_percent: float = 0
    memory_used: int = 0
    memory_total: int = 0
    disk_percent: float = 0
    disk_read_bytes: int = 0
    disk_write_bytes: int = 0
    net_bytes_sent: int = 0
    net_bytes_recv: int = 0
    load_1m: float = 0
    load_5m: float = 0
    load_15m: float = 0
    process_count: int = 0
    uptime_seconds: int = 0


class MetricHistoryPoint(BaseModel):
    time: str
    value: float


class MetricHistoryResponse(BaseModel):
    metric: str
    range: str
    data: list[MetricHistoryPoint]
