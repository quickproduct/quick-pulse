from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException

from app.core.logging import get_logger
from app.domain.entities.user import User
from app.domain.services.alert_service import AlertService
from app.schemas.alert import (
    AlertResponse,
    AlertRuleCreate,
    AlertRuleResponse,
    AlertRuleUpdate,
)
from app.utils.deps import (
    get_alert_repo,
    get_alert_rule_repo,
    get_current_user,
    require_admin,
)

router = APIRouter()
rules_router = APIRouter()
logger = get_logger("api.alerts")


@router.get("")
async def list_alerts(
    current_user: User = Depends(get_current_user),
    alert_repo=Depends(get_alert_repo),
    alert_rule_repo=Depends(get_alert_rule_repo),
):
    logger.info("alerts_list", user_id=str(current_user.id))
    try:
        service = AlertService(alert_repo, alert_rule_repo)
        alerts = await service.list_active_alerts()
        return [
            AlertResponse(
                id=a.id,
                rule_id=a.rule_id,
                severity=a.severity.value if hasattr(a.severity, "value") else str(a.severity),
                message=a.message,
                acknowledged=a.acknowledged,
                created_at=a.created_at,
            )
            for a in alerts
        ]
    except Exception as e:
        logger.error("alerts_list_error", error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to load alerts")


@router.post("/{alert_id}/acknowledge", response_model=AlertResponse)
async def acknowledge_alert(
    alert_id: UUID,
    current_user: User = Depends(require_admin),
    alert_repo=Depends(get_alert_repo),
    alert_rule_repo=Depends(get_alert_rule_repo),
):
    logger.info("alert_acknowledge", alert_id=str(alert_id), user_id=str(current_user.id))
    try:
        service = AlertService(alert_repo, alert_rule_repo)
        alert = await service.acknowledge_alert(alert_id)
        if not alert:
            raise HTTPException(status_code=404, detail="Alert not found")
        logger.info("alert_acknowledged", alert_id=str(alert_id))
        return AlertResponse(
            id=alert.id,
            rule_id=alert.rule_id,
            severity=alert.severity.value if hasattr(alert.severity, "value") else str(alert.severity),
            message=alert.message,
            acknowledged=alert.acknowledged,
            created_at=alert.created_at,
        )
    except HTTPException:
        raise
    except Exception as e:
        logger.error("alert_acknowledge_error", alert_id=str(alert_id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to acknowledge alert")


@rules_router.get("")
async def list_rules(
    current_user: User = Depends(get_current_user),
    alert_repo=Depends(get_alert_repo),
    alert_rule_repo=Depends(get_alert_rule_repo),
):
    logger.info("alert_rules_list", user_id=str(current_user.id))
    try:
        service = AlertService(alert_repo, alert_rule_repo)
        rules = await service.list_rules()
        return [
            AlertRuleResponse(
                id=r.id,
                metric_type=r.metric_type,
                threshold=r.threshold,
                operator=r.operator,
                duration_seconds=r.duration_seconds,
                enabled=r.enabled,
                created_at=r.created_at,
            )
            for r in rules
        ]
    except Exception as e:
        logger.error("alert_rules_list_error", error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to load alert rules")


@rules_router.post("", response_model=AlertRuleResponse, status_code=201)
async def create_rule(
    body: AlertRuleCreate,
    current_user: User = Depends(require_admin),
    alert_repo=Depends(get_alert_repo),
    alert_rule_repo=Depends(get_alert_rule_repo),
):
    logger.info("alert_rule_create", metric_type=body.metric_type, user_id=str(current_user.id))
    try:
        service = AlertService(alert_repo, alert_rule_repo)
        rule = await service.create_rule(body.metric_type, body.threshold, body.operator, body.duration_seconds)
        logger.info("alert_rule_created", rule_id=str(rule.id))
        return AlertRuleResponse(
            id=rule.id,
            metric_type=rule.metric_type,
            threshold=rule.threshold,
            operator=rule.operator,
            duration_seconds=rule.duration_seconds,
            enabled=rule.enabled,
            created_at=rule.created_at,
        )
    except Exception as e:
        logger.error("alert_rule_create_error", error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to create alert rule")


@rules_router.put("/{rule_id}", response_model=AlertRuleResponse)
async def update_rule(
    rule_id: UUID,
    body: AlertRuleUpdate,
    current_user: User = Depends(require_admin),
    alert_repo=Depends(get_alert_repo),
    alert_rule_repo=Depends(get_alert_rule_repo),
):
    logger.info("alert_rule_update", rule_id=str(rule_id), user_id=str(current_user.id))
    try:
        service = AlertService(alert_repo, alert_rule_repo)
        updates = body.model_dump(exclude_none=True)
        rule = await service.update_rule(rule_id, **updates)
        if not rule:
            raise HTTPException(status_code=404, detail="Rule not found")
        logger.info("alert_rule_updated", rule_id=str(rule_id))
        return AlertRuleResponse(
            id=rule.id,
            metric_type=rule.metric_type,
            threshold=rule.threshold,
            operator=rule.operator,
            duration_seconds=rule.duration_seconds,
            enabled=rule.enabled,
            created_at=rule.created_at,
        )
    except HTTPException:
        raise
    except Exception as e:
        logger.error("alert_rule_update_error", rule_id=str(rule_id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to update alert rule")


@rules_router.delete("/{rule_id}")
async def delete_rule(
    rule_id: UUID,
    current_user: User = Depends(require_admin),
    alert_repo=Depends(get_alert_repo),
    alert_rule_repo=Depends(get_alert_rule_repo),
):
    logger.info("alert_rule_delete", rule_id=str(rule_id), user_id=str(current_user.id))
    try:
        service = AlertService(alert_repo, alert_rule_repo)
        deleted = await service.delete_rule(rule_id)
        if not deleted:
            raise HTTPException(status_code=404, detail="Rule not found")
        logger.info("alert_rule_deleted", rule_id=str(rule_id))
        return {"message": "Rule deleted"}
    except HTTPException:
        raise
    except Exception as e:
        logger.error("alert_rule_delete_error", rule_id=str(rule_id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to delete alert rule")
