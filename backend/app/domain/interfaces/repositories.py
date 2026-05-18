from abc import ABC, abstractmethod
from datetime import datetime
from uuid import UUID

from app.domain.entities.alert import Alert, AlertRule
from app.domain.entities.container import Container
from app.domain.entities.event import ContainerEvent
from app.domain.entities.host import Host, HostMetric
from app.domain.entities.stack import ComposeService, ComposeStack
from app.domain.entities.user import User


class UserRepository(ABC):
    @abstractmethod
    async def get_by_id(self, user_id: UUID) -> User | None: ...

    @abstractmethod
    async def get_by_email(self, email: str) -> User | None: ...

    @abstractmethod
    async def create(self, email: str, hashed_password: str, role: str = "admin") -> User: ...

    @abstractmethod
    async def update_password(self, user_id: UUID, hashed_password: str) -> None: ...


class HostRepository(ABC):
    @abstractmethod
    async def get_by_hostname(self, hostname: str) -> Host | None: ...

    @abstractmethod
    async def get_current(self) -> Host | None: ...

    @abstractmethod
    async def upsert(self, host: Host) -> Host: ...


class MetricsRepository(ABC):
    @abstractmethod
    async def insert(self, metric: HostMetric) -> None: ...

    @abstractmethod
    async def get_latest(self, host_id: UUID) -> HostMetric | None: ...

    @abstractmethod
    async def get_history(self, host_id: UUID, metric: str, start: datetime, end: datetime) -> list[dict]: ...


class ContainerRepository(ABC):
    @abstractmethod
    async def get_by_docker_id(self, docker_id: str) -> Container | None: ...

    @abstractmethod
    async def upsert(self, container: Container) -> Container: ...

    @abstractmethod
    async def upsert_batch(self, containers: list[Container]) -> list[Container]: ...

    @abstractmethod
    async def remove_stale(self, active_docker_ids: list[str]) -> int: ...


class ContainerEventRepository(ABC):
    @abstractmethod
    async def insert(self, event: ContainerEvent) -> ContainerEvent: ...

    @abstractmethod
    async def get_recent(self, limit: int = 50) -> list[ContainerEvent]: ...


class StackRepository(ABC):
    @abstractmethod
    async def get_by_name(self, name: str) -> ComposeStack | None: ...

    @abstractmethod
    async def list_all(self) -> list[ComposeStack]: ...

    @abstractmethod
    async def upsert(self, stack: ComposeStack) -> ComposeStack: ...

    @abstractmethod
    async def upsert_services(self, stack_id: UUID, services: list[ComposeService]) -> list[ComposeService]: ...

    @abstractmethod
    async def delete(self, name: str) -> None: ...


class AlertRuleRepository(ABC):
    @abstractmethod
    async def list_all(self) -> list[AlertRule]: ...

    @abstractmethod
    async def get_enabled(self) -> list[AlertRule]: ...

    @abstractmethod
    async def get_by_id(self, rule_id: UUID) -> AlertRule | None: ...

    @abstractmethod
    async def create(self, metric_type: str, threshold: float, operator: str, duration_seconds: int) -> AlertRule: ...

    @abstractmethod
    async def update(self, rule_id: UUID, **kwargs) -> AlertRule | None: ...

    @abstractmethod
    async def delete(self, rule_id: UUID) -> bool: ...


class AlertRepository(ABC):
    @abstractmethod
    async def list_active(self, limit: int = 50) -> list[Alert]: ...

    @abstractmethod
    async def create(self, rule_id: UUID | None, severity: str, message: str) -> Alert: ...

    @abstractmethod
    async def acknowledge(self, alert_id: UUID) -> Alert | None: ...


class AuditLogRepository(ABC):
    @abstractmethod
    async def log(self, user_id: UUID | None, action: str, resource_type: str, resource_id: str | None = None, details: dict | None = None) -> None: ...

    @abstractmethod
    async def get_recent(self, limit: int = 100) -> list[dict]: ...


class SessionRepository(ABC):
    @abstractmethod
    async def create(self, user_id: UUID, refresh_token: str, expires_at: datetime) -> None: ...

    @abstractmethod
    async def get_by_refresh_token(self, refresh_token: str) -> dict | None: ...

    @abstractmethod
    async def delete_by_refresh_token(self, refresh_token: str) -> bool: ...

    @abstractmethod
    async def delete_by_user_id(self, user_id: UUID) -> int: ...
