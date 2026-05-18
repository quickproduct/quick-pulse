import pytest
from unittest.mock import AsyncMock, MagicMock

from app.core.constants import ContainerStatus
from app.domain.services.docker_service import DockerService


@pytest.mark.asyncio
async def test_list_containers():
    mock_docker = MagicMock()
    mock_container = MagicMock()
    mock_container.id = "abc123def456"
    mock_container._container = {
        "Names": ["/my-container"],
        "Image": "nginx:latest",
        "State": "running",
        "Status": "Up 2 hours",
        "Ports": [],
    }
    mock_docker.containers.list = AsyncMock(return_value=[mock_container])

    service = DockerService(mock_docker, None, None)
    result = await service.list_containers()

    assert len(result) == 1
    assert result[0]["name"] == "my-container"
    assert result[0]["image"] == "nginx:latest"
    assert result[0]["status"] == "running"


@pytest.mark.asyncio
async def test_start_container():
    mock_docker = MagicMock()
    mock_container = MagicMock()
    mock_container.start = AsyncMock()
    mock_docker.containers.container = MagicMock(return_value=mock_container)

    mock_audit = AsyncMock()
    mock_audit.log = AsyncMock()

    service = DockerService(mock_docker, None, mock_audit)
    result = await service.start_container("abc123", user_id=None)

    assert result["success"] is True
    assert "abc123" in result["message"]
    mock_container.start.assert_called_once()


@pytest.mark.asyncio
async def test_stop_container():
    mock_docker = MagicMock()
    mock_container = MagicMock()
    mock_container.stop = AsyncMock()
    mock_docker.containers.container = MagicMock(return_value=mock_container)

    service = DockerService(mock_docker, None, None)
    result = await service.stop_container("abc123")

    assert result["success"] is True
    mock_container.stop.assert_called_once()


@pytest.mark.asyncio
async def test_restart_container():
    mock_docker = MagicMock()
    mock_container = MagicMock()
    mock_container.restart = AsyncMock()
    mock_docker.containers.container = MagicMock(return_value=mock_container)

    service = DockerService(mock_docker, None, None)
    result = await service.restart_container("abc123")

    assert result["success"] is True
    mock_container.restart.assert_called_once()


@pytest.mark.asyncio
async def test_normalize_status():
    mock_docker = MagicMock()
    service = DockerService(mock_docker, None, None)

    assert service._normalize_status("running") == "running"
    assert service._normalize_status("exited") == "exited"
    assert service._normalize_status("unknown_state") == "unknown"
