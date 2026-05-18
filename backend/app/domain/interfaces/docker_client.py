from abc import ABC, abstractmethod
from typing import Any


class DockerClientInterface(ABC):
    @abstractmethod
    async def list_containers(self, all: bool = False) -> list[dict[str, Any]]: ...

    @abstractmethod
    async def inspect_container(self, container_id: str) -> dict[str, Any]: ...

    @abstractmethod
    async def start_container(self, container_id: str) -> None: ...

    @abstractmethod
    async def stop_container(self, container_id: str, timeout: int = 10) -> None: ...

    @abstractmethod
    async def restart_container(self, container_id: str, timeout: int = 10) -> None: ...

    @abstractmethod
    async def kill_container(self, container_id: str) -> None: ...

    @abstractmethod
    async def container_logs(self, container_id: str, tail: int = 100, follow: bool = False, since: str | None = None) -> Any: ...

    @abstractmethod
    async def container_stats(self, container_id: str) -> dict[str, Any]: ...

    @abstractmethod
    async def events(self) -> Any: ...

    @abstractmethod
    async def list_compose_services(self) -> list[dict[str, Any]]: ...
