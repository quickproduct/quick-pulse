import pytest
from unittest.mock import AsyncMock
from uuid import uuid4
from datetime import datetime, timezone

from app.core.constants import AlertOperator, AlertSeverity
from app.domain.services.alert_service import AlertService
from app.domain.entities.alert import Alert, AlertRule


@pytest.mark.asyncio
async def test_create_alert_rule():
    mock_alert_repo = AsyncMock()
    mock_rule_repo = AsyncMock()
    rule_id = uuid4()
    mock_rule_repo.create = AsyncMock(return_value=AlertRule(
        id=rule_id, metric_type="cpu", threshold=90.0,
        operator="gte", duration_seconds=60, enabled=True,
        created_at=datetime.now(timezone.utc),
    ))

    service = AlertService(mock_alert_repo, mock_rule_repo)
    rule = await service.create_rule("cpu", 90.0, "gte", 60)

    assert rule.metric_type == "cpu"
    assert rule.threshold == 90.0
    mock_rule_repo.create.assert_called_once_with("cpu", 90.0, "gte", 60)


@pytest.mark.asyncio
async def test_evaluate_rules_triggers_alert():
    mock_alert_repo = AsyncMock()
    mock_rule_repo = AsyncMock()

    rule_id = uuid4()
    mock_rule_repo.get_enabled = AsyncMock(return_value=[
        AlertRule(
            id=rule_id, metric_type="cpu", threshold=80.0,
            operator="gte", duration_seconds=60, enabled=True,
        )
    ])

    alert_id = uuid4()
    mock_alert_repo.create = AsyncMock(return_value=Alert(
        id=alert_id, rule_id=rule_id, severity="warning",
        message="cpu is 95.0%", acknowledged=False,
        created_at=datetime.now(timezone.utc),
    ))

    service = AlertService(mock_alert_repo, mock_rule_repo)
    alerts = await service.evaluate_rules({"cpu_percent": 95.0})

    assert len(alerts) == 1
    mock_alert_repo.create.assert_called_once()


@pytest.mark.asyncio
async def test_evaluate_rules_no_trigger():
    mock_alert_repo = AsyncMock()
    mock_rule_repo = AsyncMock()

    rule_id = uuid4()
    mock_rule_repo.get_enabled = AsyncMock(return_value=[
        AlertRule(
            id=rule_id, metric_type="cpu", threshold=90.0,
            operator="gte", duration_seconds=60, enabled=True,
        )
    ])

    service = AlertService(mock_alert_repo, mock_rule_repo)
    alerts = await service.evaluate_rules({"cpu_percent": 50.0})

    assert len(alerts) == 0
    mock_alert_repo.create.assert_not_called()


@pytest.mark.asyncio
async def test_acknowledge_alert():
    mock_alert_repo = AsyncMock()
    mock_rule_repo = AsyncMock()

    alert_id = uuid4()
    mock_alert_repo.acknowledge = AsyncMock(return_value=Alert(
        id=alert_id, severity="warning", message="test",
        acknowledged=True, created_at=datetime.now(timezone.utc),
    ))

    service = AlertService(mock_alert_repo, mock_rule_repo)
    result = await service.acknowledge_alert(alert_id)

    assert result.acknowledged is True
    mock_alert_repo.acknowledge.assert_called_once_with(alert_id)
