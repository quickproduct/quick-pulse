import operator as op
from uuid import UUID

from app.core.constants import AlertOperator, AlertSeverity
from app.core.logging import get_logger
from app.domain.entities.alert import Alert, AlertRule
from app.domain.interfaces.repositories import AlertRepository, AlertRuleRepository

logger = get_logger("alert_service")

_OPERATOR_FUNCS = {
    AlertOperator.GT: op.gt,
    AlertOperator.GTE: op.ge,
    AlertOperator.LT: op.lt,
    AlertOperator.LTE: op.le,
    AlertOperator.EQ: op.eq,
}


class AlertService:
    def __init__(self, alert_repo: AlertRepository, alert_rule_repo: AlertRuleRepository):
        self._alert_repo = alert_repo
        self._alert_rule_repo = alert_rule_repo

    async def list_rules(self) -> list[AlertRule]:
        return await self._alert_rule_repo.list_all()

    async def create_rule(self, metric_type: str, threshold: float, operator: str, duration_seconds: int) -> AlertRule:
        return await self._alert_rule_repo.create(metric_type, threshold, operator, duration_seconds)

    async def update_rule(self, rule_id: UUID, **kwargs) -> AlertRule | None:
        return await self._alert_rule_repo.update(rule_id, **kwargs)

    async def delete_rule(self, rule_id: UUID) -> bool:
        return await self._alert_rule_repo.delete(rule_id)

    async def list_active_alerts(self, limit: int = 50) -> list[Alert]:
        return await self._alert_repo.list_active(limit)

    async def acknowledge_alert(self, alert_id: UUID) -> Alert | None:
        return await self._alert_repo.acknowledge(alert_id)

    async def evaluate_rules(self, current_metrics: dict) -> list[Alert]:
        rules = await self._alert_rule_repo.get_enabled()
        new_alerts = []
        for rule in rules:
            metric_col_map = {
                "cpu": "cpu_percent",
                "memory": "memory_percent",
                "disk": "disk_percent",
                "load": "load_1m",
            }
            col = metric_col_map.get(rule.metric_type)
            if not col:
                logger.warning("alert_unknown_metric_type", metric_type=rule.metric_type, rule_id=str(rule.id))
                continue

            value = current_metrics.get(col)
            if value is None:
                continue

            if not isinstance(value, (int, float)):
                logger.warning("alert_non_numeric_metric", col=col, value_type=type(value).__name__)
                continue

            try:
                alert_op = AlertOperator(rule.operator)
                op_func = _OPERATOR_FUNCS.get(alert_op)
            except (ValueError, KeyError) as e:
                logger.warning("alert_invalid_operator", operator=rule.operator, rule_id=str(rule.id), error=str(e))
                continue

            if not op_func:
                logger.warning("alert_unmapped_operator", operator=rule.operator, rule_id=str(rule.id))
                continue

            try:
                if op_func(value, rule.threshold):
                    severity = AlertSeverity.CRITICAL if rule.threshold >= 90 else AlertSeverity.WARNING
                    message = f"{rule.metric_type} is {value}% (threshold: {rule.threshold}% {rule.operator})"
                    alert = await self._alert_repo.create(
                        rule_id=rule.id,
                        severity=severity.value,
                        message=message,
                    )
                    new_alerts.append(alert)
                    logger.warning("alert_triggered", rule_id=str(rule.id), metric=rule.metric_type, value=value, threshold=rule.threshold)
            except Exception as e:
                logger.error("alert_evaluate_rule_failed", rule_id=str(rule.id), error=str(e), exc_info=True)
                continue

        return new_alerts
