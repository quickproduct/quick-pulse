from datetime import datetime, timedelta, timezone
from uuid import UUID

from fastapi import HTTPException, status

from app.core.logging import get_logger
from app.domain.entities.plan import Plan
from app.domain.entities.subscription import Subscription, UsageRecord
from app.repositories.billing_repo import SqlBillingRepository

logger = get_logger("services.billing")


class BillingService:
    def __init__(self, billing_repo: SqlBillingRepository, trial_days: int = 14) -> None:
        self._repo = billing_repo
        self._trial_days = trial_days

    async def list_plans(self) -> list[Plan]:
        return await self._repo.list_plans(active_only=True)

    async def get_subscription(self, org_id: UUID) -> Subscription | None:
        return await self._repo.get_subscription(org_id)

    async def create_trial_subscription(self, org_id: UUID, plan_name: str = "free") -> Subscription:
        plan = await self._repo.get_plan_by_name(plan_name)
        if not plan:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Plan '{plan_name}' not found")

        now = datetime.now(timezone.utc)
        trial_end = now + timedelta(days=self._trial_days)
        sub = await self._repo.create_subscription(
            Subscription(
                org_id=org_id,
                plan_id=plan.id,
                status="trial",
                trial_ends_at=trial_end,
                current_period_start=now,
                current_period_end=trial_end,
            )
        )
        logger.info("subscription_created", org_id=str(org_id), plan=plan_name, status="trial")
        return sub

    async def change_plan(self, org_id: UUID, plan_name: str) -> Subscription:
        plan = await self._repo.get_plan_by_name(plan_name)
        if not plan:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Plan '{plan_name}' not found")

        sub = await self._repo.get_subscription(org_id)
        if not sub:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="No active subscription found")

        now = datetime.now(timezone.utc)
        sub.plan_id = plan.id
        sub.status = "active"
        sub.current_period_start = now
        sub.current_period_end = now + timedelta(days=30)
        sub.cancelled_at = None

        updated = await self._repo.update_subscription(sub)
        logger.info("plan_changed", org_id=str(org_id), new_plan=plan_name)
        return updated

    async def cancel_subscription(self, org_id: UUID) -> Subscription:
        sub = await self._repo.get_subscription(org_id)
        if not sub:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="No subscription found")
        if sub.status == "cancelled":
            raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail="Subscription already cancelled")

        sub.status = "cancelled"
        sub.cancelled_at = datetime.now(timezone.utc)
        updated = await self._repo.update_subscription(sub)
        logger.warning("subscription_cancelled", org_id=str(org_id))
        return updated

    async def get_current_plan(self, org_id: UUID) -> Plan | None:
        sub = await self._repo.get_subscription(org_id)
        if not sub:
            return None
        return await self._repo.get_plan_by_id(sub.plan_id)

    async def check_limit(self, org_id: UUID, metric: str, current_count: int) -> None:
        """Raise HTTP 402 if the org has exceeded the plan limit for `metric`."""
        plan = await self.get_current_plan(org_id)
        if not plan:
            return  # no subscription = no limit enforcement
        limits = {
            "hosts": plan.max_hosts,
            "users": plan.max_users,
            "containers": plan.max_containers,
        }
        limit = limits.get(metric, -1)
        if limit != -1 and current_count >= limit:
            raise HTTPException(
                status_code=status.HTTP_402_PAYMENT_REQUIRED,
                detail=f"Plan limit reached for {metric} ({current_count}/{limit}). Upgrade your plan.",
            )

    async def record_usage(self, org_id: UUID, metric_type: str, value: int) -> UsageRecord:
        record = await self._repo.record_usage(
            UsageRecord(org_id=org_id, metric_type=metric_type, value=value)
        )
        logger.info("usage_recorded", org_id=str(org_id), metric=metric_type, value=value)
        return record

    async def get_usage(
        self, org_id: UUID, period_start: datetime, period_end: datetime
    ) -> list[UsageRecord]:
        return await self._repo.get_usage(org_id, period_start, period_end)
