from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

from app.core.constants import MetricType


@dataclass
class Metric:
    time: datetime
    host_id: UUID
    cpu_percent: float = 0.0
    memory_percent: float = 0.0
    disk_percent: float = 0.0
