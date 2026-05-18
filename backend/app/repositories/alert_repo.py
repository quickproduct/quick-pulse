from datetime import datetime, timezone
from uuid import UUID

from sqlalchemy import select, update, delete
from sqlalchemy.ext.asyncio import AsyncSession

from app.domain.entities.alert import Alert, AlertRule
from app.domain.interfaces.repositories import AlertRepository, AlertRuleRepository
from app.infrastructure.db.models import AlertModel, AlertRuleModel, SessionModel


class SqlAlertRuleRepository(AlertRuleRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def list_all(self) -> list[AlertRule]:
        result = await self._session.execute(select(AlertRuleModel).order_by(AlertRuleModel.created_at))
        return [
            AlertRule(
                id=m.id, metric_type=m.metric_type, threshold=m.threshold,
                operator=m.operator, duration_seconds=m.duration_seconds,
                enabled=m.enabled, created_at=m.created_at,
            )
            for m in result.scalars().all()
        ]

    async def get_enabled(self) -> list[AlertRule]:
        result = await self._session.execute(
            select(AlertRuleModel).where(AlertRuleModel.enabled == True)
        )
        return [
            AlertRule(
                id=m.id, metric_type=m.metric_type, threshold=m.threshold,
                operator=m.operator, duration_seconds=m.duration_seconds,
                enabled=m.enabled, created_at=m.created_at,
            )
            for m in result.scalars().all()
        ]

    async def get_by_id(self, rule_id: UUID) -> AlertRule | None:
        result = await self._session.execute(select(AlertRuleModel).where(AlertRuleModel.id == rule_id))
        m = result.scalar_one_or_none()
        if not m:
            return None
        return AlertRule(
            id=m.id, metric_type=m.metric_type, threshold=m.threshold,
            operator=m.operator, duration_seconds=m.duration_seconds,
            enabled=m.enabled, created_at=m.created_at,
        )

    async def create(self, metric_type: str, threshold: float, operator: str, duration_seconds: int) -> AlertRule:
        model = AlertRuleModel(
            metric_type=metric_type, threshold=threshold,
            operator=operator, duration_seconds=duration_seconds,
        )
        self._session.add(model)
        await self._session.flush()
        return AlertRule(
            id=model.id, metric_type=model.metric_type, threshold=model.threshold,
            operator=model.operator, duration_seconds=model.duration_seconds,
            enabled=model.enabled, created_at=model.created_at,
        )

    async def update(self, rule_id: UUID, **kwargs) -> AlertRule | None:
        await self._session.execute(update(AlertRuleModel).where(AlertRuleModel.id == rule_id).values(**kwargs))
        return await self.get_by_id(rule_id)

    async def delete(self, rule_id: UUID) -> bool:
        result = await self._session.execute(delete(AlertRuleModel).where(AlertRuleModel.id == rule_id))
        return result.rowcount > 0


class SqlAlertRepository(AlertRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def list_active(self, limit: int = 50) -> list[Alert]:
        result = await self._session.execute(
            select(AlertModel)
            .where(AlertModel.acknowledged == False)
            .order_by(AlertModel.created_at.desc())
            .limit(limit)
        )
        return [
            Alert(
                id=m.id, rule_id=m.rule_id, severity=m.severity,
                message=m.message, acknowledged=m.acknowledged, created_at=m.created_at,
            )
            for m in result.scalars().all()
        ]

    async def create(self, rule_id: UUID | None, severity: str, message: str) -> Alert:
        model = AlertModel(rule_id=rule_id, severity=severity, message=message)
        self._session.add(model)
        await self._session.flush()
        return Alert(
            id=model.id, rule_id=model.rule_id, severity=model.severity,
            message=model.message, acknowledged=model.acknowledged, created_at=model.created_at,
        )

    async def acknowledge(self, alert_id: UUID) -> Alert | None:
        await self._session.execute(
            update(AlertModel).where(AlertModel.id == alert_id).values(acknowledged=True)
        )
        result = await self._session.execute(select(AlertModel).where(AlertModel.id == alert_id))
        m = result.scalar_one_or_none()
        if not m:
            return None
        return Alert(
            id=m.id, rule_id=m.rule_id, severity=m.severity,
            message=m.message, acknowledged=m.acknowledged, created_at=m.created_at,
        )


class SqlSessionRepository:
    def __init__(self, session: AsyncSession):
        self._session = session

    async def create(self, user_id: UUID, refresh_token: str, expires_at: datetime) -> None:
        model = SessionModel(user_id=user_id, refresh_token=refresh_token, expires_at=expires_at)
        self._session.add(model)
        await self._session.flush()

    async def get_by_refresh_token(self, refresh_token: str) -> dict | None:
        result = await self._session.execute(
            select(SessionModel).where(SessionModel.refresh_token == refresh_token)
        )
        m = result.scalar_one_or_none()
        if not m:
            return None
        return {"id": m.id, "user_id": m.user_id, "expires_at": m.expires_at}

    async def delete_by_refresh_token(self, refresh_token: str) -> bool:
        result = await self._session.execute(
            delete(SessionModel).where(SessionModel.refresh_token == refresh_token)
        )
        return result.rowcount > 0

    async def delete_by_user_id(self, user_id: UUID) -> int:
        result = await self._session.execute(
            delete(SessionModel).where(SessionModel.user_id == user_id)
        )
        return result.rowcount
