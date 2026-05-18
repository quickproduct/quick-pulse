from dataclasses import dataclass
from datetime import datetime
from uuid import UUID


@dataclass
class Host:
    id: UUID
    hostname: str
    ip_address: str
    os_info: str | None = None
    cpu_count: int = 0
    total_memory: int = 0
    total_disk: int = 0
    created_at: datetime | None = None


@dataclass
class HostMetric:
    time: datetime
    host_id: UUID
    cpu_percent: float = 0.0
    memory_percent: float = 0.0
    memory_used: int = 0
    memory_total: int = 0
    disk_percent: float = 0.0
    disk_read_bytes: int = 0
    disk_write_bytes: int = 0
    net_bytes_sent: int = 0
    net_bytes_recv: int = 0
    load_1m: float = 0.0
    load_5m: float = 0.0
    load_15m: float = 0.0
    process_count: int = 0
    uptime_seconds: int = 0
