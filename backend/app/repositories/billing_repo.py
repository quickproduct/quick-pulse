from datetime import datetime
from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.domain.entities.plan import Plan
from app.domain.entities.subscription import ApiKey, Subscription, UsageRecord
from app.infrastructure.db.models import ApiKeyModel, PlanModel, SubscriptionModel, UsageRecordModel


def _plan_from_model(m: PlanModel) -> Plan:
    return Plan(
        id=m.id,
        name=m.name,
        display_name=m.display_name,
        max_hosts=m.max_hosts,
        max_users=m.max_users,
        max_containers=m.max_containers,
        metrics_retention_days=m.metrics_retention_days,
        price_monthly_cents=m.price_monthly_cents,
        price_yearly_cents=m.price_yearly_cents,
        features=m.features,
        is_active=m.is_active,
        created_at=m.created_at,
    )


def _sub_from_model(m: SubscriptionModel) -> Subscription:
    return Subscription(
        id=m.id,
        org_id=m.org_id,
        plan_id=m.plan_id,
        status=m.status,
        trial_ends_at=m.trial_ends_at,
        current_period_start=m.current_period_start,
        current_period_end=m.current_period_end,
        stripe_customer_id=m.stripe_customer_id,
        stripe_subscription_id=m.stripe_subscription_id,
        cancelled_at=m.cancelled_at,
        created_at=m.created_at,
        updated_at=m.updated_at,
    )


def _apikey_from_model(m: ApiKeyModel) -> ApiKey:
    return ApiKey(
        id=m.id,
        org_id=m.org_id,
        user_id=m.user_id,
        name=m.name,
        key_hash=m.key_hash,
        key_prefix=m.key_prefix,
        scopes=m.scopes,
        is_active=m.is_active,
        last_used_at=m.last_used_at,
        expires_at=m.expires_at,
        created_at=m.created_at,
    )


class SqlBillingRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    # ── Plans ────────────────────────────────────────────────────────────────

    async def list_plans(self, active_only: bool = True) -> list[Plan]:
        q = select(PlanModel)
        if active_only:
            q = q.where(PlanModel.is_active.is_(True))
        result = await self._session.execute(q)
        return [_plan_from_model(m) for m in result.scalars().all()]

    async def get_plan_by_name(self, name: str) -> Plan | None:
        result = await self._session.execute(
            select(PlanModel).where(PlanModel.name == name)
        )
        model = result.scalar_one_or_none()
        return _plan_from_model(model) if model else None

    async def get_plan_by_id(self, plan_id: UUID) -> Plan | None:
        result = await self._session.execute(
            select(PlanModel).where(PlanModel.id == plan_id)
        )
        model = result.scalar_one_or_none()
        return _plan_from_model(model) if model else None

    async def create_plan(self, plan: Plan) -> Plan:
        model = PlanModel(
            name=plan.name,
            display_name=plan.display_name,
            max_hosts=plan.max_hosts,
            max_users=plan.max_users,
            max_containers=plan.max_containers,
            metrics_retention_days=plan.metrics_retention_days,
            price_monthly_cents=plan.price_monthly_cents,
            price_yearly_cents=plan.price_yearly_cents,
            features=plan.features,
            is_active=plan.is_active,
        )
        self._session.add(model)
        await self._session.flush()
        return _plan_from_model(model)

    # ── Subscriptions ─────────────────────────────────────────────────────────

    async def get_subscription(self, org_id: UUID) -> Subscription | None:
        result = await self._session.execute(
            select(SubscriptionModel).where(SubscriptionModel.org_id == org_id)
        )
        model = result.scalar_one_or_none()
        return _sub_from_model(model) if model else None

    async def create_subscription(self, sub: Subscription) -> Subscription:
        model = SubscriptionModel(
            org_id=sub.org_id,
            plan_id=sub.plan_id,
            status=sub.status,
            trial_ends_at=sub.trial_ends_at,
            current_period_start=sub.current_period_start,
            current_period_end=sub.current_period_end,
        )
        self._session.add(model)
        await self._session.flush()
        return _sub_from_model(model)

    async def update_subscription(self, sub: Subscription) -> Subscription:
        result = await self._session.execute(
            select(SubscriptionModel).where(SubscriptionModel.id == sub.id)
        )
        model = result.scalar_one()
        model.plan_id = sub.plan_id
        model.status = sub.status
        model.trial_ends_at = sub.trial_ends_at
        model.current_period_start = sub.current_period_start
        model.current_period_end = sub.current_period_end
        model.stripe_customer_id = sub.stripe_customer_id
        model.stripe_subscription_id = sub.stripe_subscription_id
        model.cancelled_at = sub.cancelled_at
        await self._session.flush()
        return _sub_from_model(model)

    # ── Usage Records ─────────────────────────────────────────────────────────

    async def record_usage(self, record: UsageRecord) -> UsageRecord:
        model = UsageRecordModel(
            org_id=record.org_id,
            metric_type=record.metric_type,
            value=record.value,
        )
        self._session.add(model)
        await self._session.flush()
        return UsageRecord(
            id=model.id,
            org_id=model.org_id,
            metric_type=model.metric_type,
            value=model.value,
            recorded_at=model.recorded_at,
        )

    async def get_usage(
        self, org_id: UUID, period_start: datetime, period_end: datetime
    ) -> list[UsageRecord]:
        result = await self._session.execute(
            select(UsageRecordModel).where(
                UsageRecordModel.org_id == org_id,
                UsageRecordModel.recorded_at >= period_start,
                UsageRecordModel.recorded_at <= period_end,
            )
        )
        return [
            UsageRecord(
                id=m.id,
                org_id=m.org_id,
                metric_type=m.metric_type,
                value=m.value,
                recorded_at=m.recorded_at,
            )
            for m in result.scalars().all()
        ]

    # ── API Keys ──────────────────────────────────────────────────────────────

    async def create_api_key(self, key: ApiKey) -> ApiKey:
        model = ApiKeyModel(
            org_id=key.org_id,
            user_id=key.user_id,
            name=key.name,
            key_hash=key.key_hash,
            key_prefix=key.key_prefix,
            scopes=key.scopes,
            expires_at=key.expires_at,
        )
        self._session.add(model)
        await self._session.flush()
        return _apikey_from_model(model)

    async def get_api_key_by_hash(self, key_hash: str) -> ApiKey | None:
        result = await self._session.execute(
            select(ApiKeyModel).where(
                ApiKeyModel.key_hash == key_hash,
                ApiKeyModel.is_active.is_(True),
            )
        )
        model = result.scalar_one_or_none()
        return _apikey_from_model(model) if model else None

    async def list_api_keys(self, org_id: UUID, user_id: UUID) -> list[ApiKey]:
        result = await self._session.execute(
            select(ApiKeyModel).where(
                ApiKeyModel.org_id == org_id,
                ApiKeyModel.user_id == user_id,
            )
        )
        return [_apikey_from_model(m) for m in result.scalars().all()]

    async def revoke_api_key(self, key_id: UUID) -> None:
        result = await self._session.execute(
            select(ApiKeyModel).where(ApiKeyModel.id == key_id)
        )
        model = result.scalar_one_or_none()
        if model is None:
            raise ValueError(f"API key {key_id} not found")
        model.is_active = False
        await self._session.flush()

    async def touch_api_key(self, key_id: UUID, used_at: datetime) -> None:
        result = await self._session.execute(
            select(ApiKeyModel).where(ApiKeyModel.id == key_id)
        )
        model = result.scalar_one_or_none()
        if model:
            model.last_used_at = used_at
            await self._session.flush()
