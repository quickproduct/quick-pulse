import asyncio
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest
from httpx import ASGITransport, AsyncClient

from app.core.security import create_access_token, hash_password
from app.infrastructure.db.models import UserModel


@pytest.fixture
def mock_db_session():
    session = AsyncMock()
    session.commit = AsyncMock()
    session.rollback = AsyncMock()
    session.flush = AsyncMock()
    session.execute = AsyncMock()
    session.add = MagicMock()
    return session


@pytest.fixture
def mock_redis():
    redis = AsyncMock()
    redis.get = AsyncMock(return_value=None)
    redis.set = AsyncMock(return_value=True)
    redis.publish = AsyncMock()
    redis.aclose = AsyncMock()
    return redis


@pytest.fixture
def mock_docker():
    docker = MagicMock()
    docker.containers = MagicMock()
    docker.containers.list = AsyncMock(return_value=[])
    docker.containers.container = MagicMock()
    docker.close = AsyncMock()
    return docker


@pytest.fixture
def sample_user_id():
    return uuid4()


@pytest.fixture
def sample_user(sample_user_id):
    return UserModel(
        id=sample_user_id,
        email="test@quickpulse.local",
        hashed_password=hash_password("testpassword"),
        role="admin",
        is_active=True,
    )


@pytest.fixture
def auth_token(sample_user_id):
    return create_access_token({"sub": str(sample_user_id)})


@pytest.fixture
def auth_headers(auth_token):
    return {"Authorization": f"Bearer {auth_token}"}
