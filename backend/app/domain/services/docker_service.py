from typing import Any

import aiodocker

from app.core.constants import ContainerStatus
from app.core.logging import get_logger
from app.domain.entities.container import Container
from app.domain.interfaces.repositories import AuditLogRepository, ContainerRepository

logger = get_logger("docker_service")


class DockerService:
    def __init__(self, docker: Any, container_repo: ContainerRepository | None, audit_repo: AuditLogRepository):
        self._docker = docker
        self._container_repo = container_repo
        self._audit_repo = audit_repo

    async def list_containers(self, all: bool = False) -> list[dict]:
        try:
            containers = await self._docker.containers.list(all=all)
        except aiodocker.DockerError as e:
            logger.error("docker_list_containers_failed", error=str(e), exc_info=True)
            raise
        except Exception as e:
            logger.error("docker_list_containers_error", error=str(e), exc_info=True)
            raise

        result = []
        for c in containers:
            try:
                raw = c._container or {}
                labels = raw.get("Labels") or {}
                project = labels.get("com.docker.compose.project", "")
                
                names = raw.get("Names") or []
                name = names[0].lstrip("/") if names else (c.id or "")[:12]

                # Filter out QuickPulse's own infrastructure
                if name.startswith("qp-") or project == "quickpulse":
                    continue
                info = {
                    "docker_id": (c.id or "")[:12],
                    "name": name,
                    "image": raw.get("Image", ""),
                    "status": self._normalize_status(raw.get("State", "unknown")),
                    "ports": raw.get("Ports") or [],
                    "state": raw.get("State", ""),
                    "status_text": raw.get("Status", ""),
                }
                result.append(info)
            except Exception as e:
                logger.warning("docker_container_parse_error", error=str(e))
                continue
        return result

    async def inspect_container(self, container_id: str) -> dict:
        try:
            container = self._docker.containers.container(container_id)
            info = await container.show()
            return info
        except aiodocker.DockerError as e:
            status = getattr(e, "status", None)
            if status == 404:
                logger.warning("container_not_found", container_id=container_id)
                raise ValueError(f"Container {container_id} not found") from e
            logger.error("docker_inspect_failed", container_id=container_id, error=str(e), exc_info=True)
            raise
        except Exception as e:
            logger.error("docker_inspect_error", container_id=container_id, error=str(e), exc_info=True)
            raise

    async def start_container(self, container_id: str, user_id=None) -> dict:
        try:
            container = self._docker.containers.container(container_id)
            await container.start()
            if user_id and self._audit_repo:
                await self._audit_repo.log(user_id, "start", "container", container_id)
            logger.info("container_started", container_id=container_id)
            return {"success": True, "message": f"Container {container_id} started", "container_id": container_id}
        except aiodocker.DockerError as e:
            status = getattr(e, "status", None)
            if status == 404:
                raise ValueError(f"Container {container_id} not found") from e
            logger.error("container_start_failed", container_id=container_id, error=str(e), exc_info=True)
            raise
        except Exception as e:
            logger.error("container_start_error", container_id=container_id, error=str(e), exc_info=True)
            raise

    async def stop_container(self, container_id: str, user_id=None) -> dict:
        try:
            container = self._docker.containers.container(container_id)
            await container.stop()
            if user_id and self._audit_repo:
                await self._audit_repo.log(user_id, "stop", "container", container_id)
            logger.info("container_stopped", container_id=container_id)
            return {"success": True, "message": f"Container {container_id} stopped", "container_id": container_id}
        except aiodocker.DockerError as e:
            status = getattr(e, "status", None)
            if status == 404:
                raise ValueError(f"Container {container_id} not found") from e
            logger.error("container_stop_failed", container_id=container_id, error=str(e), exc_info=True)
            raise
        except Exception as e:
            logger.error("container_stop_error", container_id=container_id, error=str(e), exc_info=True)
            raise

    async def restart_container(self, container_id: str, user_id=None) -> dict:
        try:
            container = self._docker.containers.container(container_id)
            await container.restart()
            if user_id and self._audit_repo:
                await self._audit_repo.log(user_id, "restart", "container", container_id)
            logger.info("container_restarted", container_id=container_id)
            return {"success": True, "message": f"Container {container_id} restarted", "container_id": container_id}
        except aiodocker.DockerError as e:
            status = getattr(e, "status", None)
            if status == 404:
                raise ValueError(f"Container {container_id} not found") from e
            logger.error("container_restart_failed", container_id=container_id, error=str(e), exc_info=True)
            raise
        except Exception as e:
            logger.error("container_restart_error", container_id=container_id, error=str(e), exc_info=True)
            raise

    async def container_logs(self, container_id: str, tail: int = 100, follow: bool = False, since: str | None = None) -> list[str]:
        """Return collected log lines as a list. Does NOT stream (follow=False)."""
        tail = max(1, min(tail, 500))
        try:
            container = self._docker.containers.container(container_id)
            kwargs: dict = {"tail": tail, "follow": False}
            if since:
                kwargs["since"] = since
            log_gen = container.log(stdout=True, stderr=True, **kwargs)
            # log() returns an async generator — collect all lines eagerly
            lines: list[str] = []
            async for line in log_gen:
                if isinstance(line, bytes):
                    lines.append(line.decode("utf-8", errors="replace").rstrip())
                else:
                    lines.append(str(line).rstrip())
            return lines
        except aiodocker.DockerError as e:
            status = getattr(e, "status", None)
            if status == 404:
                raise ValueError(f"Container {container_id} not found") from e
            logger.error("container_logs_failed", container_id=container_id, error=str(e), exc_info=True)
            raise
        except Exception as e:
            logger.error("container_logs_error", container_id=container_id, error=str(e), exc_info=True)
            raise

    async def container_stats(self, container_id: str) -> dict:
        try:
            container = self._docker.containers.container(container_id)
            stats = await container.stats(stream=False)
            return stats
        except aiodocker.DockerError as e:
            logger.error("container_stats_failed", container_id=container_id, error=str(e), exc_info=True)
            return {}
        except Exception as e:
            logger.error("container_stats_error", container_id=container_id, error=str(e), exc_info=True)
            return {}

    def _normalize_status(self, state: str) -> str:
        state_lower = (state or "").lower()
        for status in ContainerStatus:
            if status.value == state_lower:
                return status.value
        return ContainerStatus.UNKNOWN.value
