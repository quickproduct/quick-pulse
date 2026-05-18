from enum import Enum


class ContainerStatus(str, Enum):
    RUNNING = "running"
    STOPPED = "stopped"
    PAUSED = "paused"
    RESTARTING = "restarting"
    DEAD = "dead"
    CREATED = "created"
    EXITED = "exited"
    UNKNOWN = "unknown"


class StackStatus(str, Enum):
    RUNNING = "running"
    STOPPED = "stopped"
    PARTIAL = "partial"
    UNKNOWN = "unknown"


class AlertSeverity(str, Enum):
    INFO = "info"
    WARNING = "warning"
    CRITICAL = "critical"


class AlertOperator(str, Enum):
    GT = "gt"
    GTE = "gte"
    LT = "lt"
    LTE = "lte"
    EQ = "eq"


class MetricType(str, Enum):
    CPU = "cpu"
    MEMORY = "memory"
    DISK = "disk"
    NETWORK = "network"
    LOAD = "load"


class EventType(str, Enum):
    CONTAINER_START = "container_start"
    CONTAINER_STOP = "container_stop"
    CONTAINER_RESTART = "container_restart"
    CONTAINER_DIE = "container_die"
    CONTAINER_HEALTH = "container_health"
    CONTAINER_CREATE = "container_create"
    CONTAINER_DESTROY = "container_destroy"
    COMPOSE_UP = "compose_up"
    COMPOSE_DOWN = "compose_down"


class UserRole(str, Enum):
    ADMIN = "admin"
    VIEWER = "viewer"


METRICS_COLLECTION_INTERVAL = 10
LOG_TAIL_DEFAULT = 100
LOG_TAIL_MAX = 5000
EVENTS_BUFFER_SIZE = 500
METRICS_CACHE_TTL = 60
LATEST_METRICS_CACHE_TTL = 10
WS_HEARTBEAT_INTERVAL = 30
LOG_TAIL_CACHE_TTL = 300

HISTORY_RANGES = {
    "1h": "1 hour",
    "24h": "24 hours",
    "7d": "7 days",
}
