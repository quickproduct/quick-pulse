from typing import Any

import aiodocker

from app.core.constants import StackStatus
from app.core.logging import get_logger
from app.domain.interfaces.repositories import AuditLogRepository, StackRepository

logger = get_logger("compose_service")


class ComposeService:
    def __init__(self, docker: Any, stack_repo: StackRepository, audit_repo: AuditLogRepository | None):
        self._docker = docker
        self._stack_repo = stack_repo
        self._audit_repo = audit_repo

    async def list_stacks(self) -> list[dict]:
        try:
            containers = await self._docker.containers.list(all=True)
        except Exception as e:
            logger.error("compose_list_containers_failed", error=str(e), exc_info=True)
            raise

        stacks: dict[str, dict] = {}
        for c in containers:
            try:
                raw = c._container or {}
                labels = raw.get("Labels") or {}
                project = labels.get("com.docker.compose.project")
                if not project or project == "quickpulse":
                    continue
                if project not in stacks:
                    stacks[project] = {
                        "name": project,
                        "project_dir": labels.get("com.docker.compose.project.working_dir", ""),
                        "services": [],
                        "running": 0,
                        "total": 0,
                    }
                state = raw.get("State", "unknown")
                names = raw.get("Names") or []
                name = names[0].lstrip("/") if names else (c.id or "")[:12]
                service_name = labels.get("com.docker.compose.service", name)
                stacks[project]["services"].append({
                    "name": service_name,
                    "container_id": (c.id or "")[:12],
                    "status": state,
                })
                stacks[project]["total"] += 1
                if state == "running":
                    stacks[project]["running"] += 1
            except Exception as e:
                logger.warning("compose_container_parse_error", error=str(e))
                continue

        result = []
        for name, data in stacks.items():
            if data["running"] == data["total"] and data["total"] > 0:
                status = StackStatus.RUNNING
            elif data["running"] > 0:
                status = StackStatus.PARTIAL
            else:
                status = StackStatus.STOPPED
            data["status"] = status.value
            data["services_count"] = data["total"]
            result.append(data)
        return result

    async def get_stack(self, name: str) -> dict | None:
        stacks = await self.list_stacks()
        for s in stacks:
            if s["name"] == name:
                return s
        return None

    async def start_stack(self, name: str, user_id=None) -> dict:
        try:
            containers = await self._docker.containers.list(
                all=True, filters={"label": f"com.docker.compose.project={name}"}
            )
        except Exception as e:
            logger.error("compose_start_list_failed", name=name, error=str(e), exc_info=True)
            raise

        errors = []
        for c in containers:
            try:
                raw = c._container or {}
                state = raw.get("State", "")
                if state != "running":
                    await c.start()
            except Exception as e:
                cid = (c.id or "")[:12]
                logger.warning("compose_start_container_failed", container=cid, stack=name, error=str(e))
                errors.append(cid)

        if user_id and self._audit_repo:
            try:
                await self._audit_repo.log(user_id, "start_stack", "stack", name)
            except Exception as e:
                logger.warning("compose_audit_log_failed", error=str(e))

        if errors:
            logger.warning("compose_start_partial", name=name, failed_containers=errors)
        else:
            logger.info("compose_stack_started", name=name)

        return {"success": len(errors) == 0, "message": f"Stack {name} started", "stack_name": name}

    async def stop_stack(self, name: str, user_id=None) -> dict:
        try:
            containers = await self._docker.containers.list(
                filters={"label": f"com.docker.compose.project={name}"}
            )
        except Exception as e:
            logger.error("compose_stop_list_failed", name=name, error=str(e), exc_info=True)
            raise

        errors = []
        for c in containers:
            try:
                await c.stop()
            except Exception as e:
                cid = (c.id or "")[:12]
                logger.warning("compose_stop_container_failed", container=cid, stack=name, error=str(e))
                errors.append(cid)

        if user_id and self._audit_repo:
            try:
                await self._audit_repo.log(user_id, "stop_stack", "stack", name)
            except Exception as e:
                logger.warning("compose_audit_log_failed", error=str(e))

        if errors:
            logger.warning("compose_stop_partial", name=name, failed_containers=errors)
        else:
            logger.info("compose_stack_stopped", name=name)

        return {"success": len(errors) == 0, "message": f"Stack {name} stopped", "stack_name": name}

    async def restart_stack(self, name: str, user_id=None) -> dict:
        try:
            containers = await self._docker.containers.list(
                filters={"label": f"com.docker.compose.project={name}"}
            )
        except Exception as e:
            logger.error("compose_restart_list_failed", name=name, error=str(e), exc_info=True)
            raise

        errors = []
        for c in containers:
            try:
                await c.restart()
            except Exception as e:
                cid = (c.id or "")[:12]
                logger.warning("compose_restart_container_failed", container=cid, stack=name, error=str(e))
                errors.append(cid)

        if user_id and self._audit_repo:
            try:
                await self._audit_repo.log(user_id, "restart_stack", "stack", name)
            except Exception as e:
                logger.warning("compose_audit_log_failed", error=str(e))

        if errors:
            logger.warning("compose_restart_partial", name=name, failed_containers=errors)
        else:
            logger.info("compose_stack_restarted", name=name)

        return {"success": len(errors) == 0, "message": f"Stack {name} restarted", "stack_name": name}
