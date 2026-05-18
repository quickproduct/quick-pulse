from uuid import UUID

from app.core.logging import get_logger
from app.domain.interfaces.repositories import AuditLogRepository

logger = get_logger("audit_service")


class AuditService:
    def __init__(self, audit_repo: AuditLogRepository):
        self._audit_repo = audit_repo

    async def log_action(
        self,
        user_id: UUID | None,
        action: str,
        resource_type: str,
        resource_id: str | None = None,
        details: dict | None = None,
    ) -> None:
        await self._audit_repo.log(user_id, action, resource_type, resource_id, details)
        logger.info("audit_log", action=action, resource_type=resource_type, resource_id=resource_id)

    async def get_recent(self, limit: int = 100) -> list[dict]:
        return await self._audit_repo.get_recent(limit)
