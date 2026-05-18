import pytest
from unittest.mock import AsyncMock
from uuid import uuid4
from datetime import datetime, timezone

from app.domain.services.audit_service import AuditService


@pytest.mark.asyncio
async def test_log_action():
    mock_audit_repo = AsyncMock()
    service = AuditService(mock_audit_repo)

    user_id = uuid4()
    await service.log_action(user_id, "start", "container", "abc123")

    mock_audit_repo.log.assert_called_once_with(
        user_id, "start", "container", "abc123", None
    )


@pytest.mark.asyncio
async def test_get_recent():
    mock_audit_repo = AsyncMock()
    mock_audit_repo.get_recent = AsyncMock(return_value=[
        {"id": "1", "action": "start", "resource_type": "container"},
    ])

    service = AuditService(mock_audit_repo)
    result = await service.get_recent()

    assert len(result) == 1
    assert result[0]["action"] == "start"
