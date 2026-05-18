from datetime import datetime, timezone

from fastapi import APIRouter, Depends, HTTPException

from app.core.config import get_settings
from app.core.logging import get_logger
from app.domain.entities.user import User
from app.domain.services.billing_service import BillingService
from app.domain.services.organization_service import OrganizationService
from app.schemas.billing import (
    CancelRequest,
    PlanResponse,
    SubscribeRequest,
    SubscriptionResponse,
    UsageItem,
    UsageResponse,
)
from app.utils.deps import get_billing_repo, get_current_user, get_org_repo, require_admin

router = APIRouter()
logger = get_logger("api.billing")


def _get_billing_svc(billing_repo=Depends(get_billing_repo)) -> BillingService:
    settings = get_settings()
    return BillingService(billing_repo, trial_days=settings.TRIAL_DAYS)


def _get_org_svc(org_repo=Depends(get_org_repo)) -> OrganizationService:
    return OrganizationService(org_repo)


@router.get("/plans", response_model=list[PlanResponse])
async def list_plans(billing_svc: BillingService = Depends(_get_billing_svc)):
    """Public — list all active pricing plans."""
    try:
        plans = await billing_svc.list_plans()
        return [
            PlanResponse(
                id=p.id,
                name=p.name,
                display_name=p.display_name,
                max_hosts=p.max_hosts,
                max_users=p.max_users,
                max_containers=p.max_containers,
                metrics_retention_days=p.metrics_retention_days,
                price_monthly_cents=p.price_monthly_cents,
                price_yearly_cents=p.price_yearly_cents,
                features=p.features,
                is_active=p.is_active,
            )
            for p in plans
        ]
    except Exception as e:
        logger.error("billing_list_plans_error", error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to list plans")


@router.get("/subscription", response_model=SubscriptionResponse | None)
async def get_subscription(
    current_user: User = Depends(require_admin),
    billing_svc: BillingService = Depends(_get_billing_svc),
    org_svc: OrganizationService = Depends(_get_org_svc),
):
    try:
        org = await org_svc.get_org_for_user(current_user.id)
        if not org:
            return None
        return await billing_svc.get_subscription(org.id)
    except Exception as e:
        logger.error("billing_get_subscription_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to get subscription")


@router.post("/subscribe", response_model=SubscriptionResponse)
async def subscribe(
    body: SubscribeRequest,
    current_user: User = Depends(require_admin),
    billing_svc: BillingService = Depends(_get_billing_svc),
    org_svc: OrganizationService = Depends(_get_org_svc),
):
    try:
        org = await org_svc.get_org_for_user(current_user.id)
        if not org:
            raise HTTPException(status_code=404, detail="No organization found")
        sub = await billing_svc.change_plan(org.id, body.plan_name)
        logger.info("subscribed", org_id=str(org.id), plan=body.plan_name, user_id=str(current_user.id))
        return sub
    except HTTPException:
        raise
    except Exception as e:
        logger.error("billing_subscribe_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to change subscription plan")


@router.post("/cancel", response_model=SubscriptionResponse)
async def cancel_subscription(
    body: CancelRequest,
    current_user: User = Depends(require_admin),
    billing_svc: BillingService = Depends(_get_billing_svc),
    org_svc: OrganizationService = Depends(_get_org_svc),
):
    try:
        org = await org_svc.get_org_for_user(current_user.id)
        if not org:
            raise HTTPException(status_code=404, detail="No organization found")
        sub = await billing_svc.cancel_subscription(org.id)
        logger.warning("subscription_cancelled_via_api", org_id=str(org.id), reason=body.reason, user_id=str(current_user.id))
        return sub
    except HTTPException:
        raise
    except Exception as e:
        logger.error("billing_cancel_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to cancel subscription")


@router.get("/usage", response_model=UsageResponse | None)
async def get_usage(
    current_user: User = Depends(require_admin),
    billing_svc: BillingService = Depends(_get_billing_svc),
    org_svc: OrganizationService = Depends(_get_org_svc),
):
    try:
        org = await org_svc.get_org_for_user(current_user.id)
        if not org:
            return None

        sub = await billing_svc.get_subscription(org.id)
        if not sub:
            return None

        records = await billing_svc.get_usage(org.id, sub.current_period_start, sub.current_period_end)
        return UsageResponse(
            org_id=org.id,
            period_start=sub.current_period_start,
            period_end=sub.current_period_end,
            usage=[UsageItem(metric_type=r.metric_type, value=r.value, recorded_at=r.recorded_at) for r in records],
        )
    except Exception as e:
        logger.error("billing_get_usage_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to get usage data")
