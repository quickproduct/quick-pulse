import pytest
from unittest.mock import AsyncMock
from uuid import uuid4
from datetime import datetime, timezone

from app.domain.services.metrics_service import MetricsService
from app.domain.entities.host import HostMetric


@pytest.mark.asyncio
async def test_store_metric():
    host_id = uuid4()
    mock_metrics_repo = AsyncMock()
    mock_host_repo = AsyncMock()

    service = MetricsService(mock_metrics_repo, mock_host_repo)

    raw = {
        "cpu_percent": 45.5,
        "memory_percent": 60.2,
        "memory_used": 8000000000,
        "memory_total": 16000000000,
        "disk_percent": 55.0,
        "disk_read_bytes": 1000000,
        "disk_write_bytes": 500000,
        "net_bytes_sent": 2000000,
        "net_bytes_recv": 3000000,
        "load_1m": 1.5,
        "load_5m": 1.2,
        "load_15m": 1.0,
        "process_count": 150,
        "uptime_seconds": 86400,
    }

    await service.store_metric(host_id, raw)
    mock_metrics_repo.insert.assert_called_once()

    call_args = mock_metrics_repo.insert.call_args[0][0]
    assert isinstance(call_args, HostMetric)
    assert call_args.cpu_percent == 45.5
    assert call_args.memory_percent == 60.2
    assert call_args.process_count == 150


@pytest.mark.asyncio
async def test_get_summary():
    host_id = uuid4()

    from app.domain.entities.host import Host
    mock_host_repo = AsyncMock()
    mock_host_repo.get_current = AsyncMock(return_value=Host(
        id=host_id, hostname="test", ip_address="127.0.0.1",
    ))

    mock_metrics_repo = AsyncMock()
    mock_metrics_repo.get_latest = AsyncMock(return_value=HostMetric(
        time=datetime.now(timezone.utc),
        host_id=host_id,
        cpu_percent=42.0,
        memory_percent=65.0,
        memory_used=8000000000,
        memory_total=16000000000,
        disk_percent=50.0,
        process_count=100,
        uptime_seconds=3600,
    ))

    service = MetricsService(mock_metrics_repo, mock_host_repo)
    summary = await service.get_summary()

    assert summary["cpu_percent"] == 42.0
    assert summary["memory_percent"] == 65.0
    assert summary["disk_percent"] == 50.0
