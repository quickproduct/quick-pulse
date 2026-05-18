from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.domain.interfaces.repositories import AuditLogRepository
from app.infrastructure.db.models import AuditLogModel


class SqlAuditLogRepository(AuditLogRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def log(self, user_id: UUID | None, action: str, resource_type: str, resource_id: str | None = None, details: dict | None = None) -> None:
        model = AuditLogModel(
            user_id=user_id, action=action, resource_type=resource_type,
            resource_id=resource_id, details=details,
        )
        self._session.add(model)
        await self._session.flush()

    async def get_recent(self, limit: int = 100) -> list[dict]:
        result = await self._session.execute(
            select(AuditLogModel).order_by(AuditLogModel.timestamp.desc()).limit(limit)
        )
        return [
            {
                "id": str(m.id),
                "user_id": str(m.user_id) if m.user_id else None,
                "action": m.action,
                "resource_type": m.resource_type,
                "resource_id": m.resource_id,
                "details": m.details,
                "timestamp": m.timestamp.isoformat() if m.timestamp else None,
            }
            for m in result.scalars().all()
        ]
