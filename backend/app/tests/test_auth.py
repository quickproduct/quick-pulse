import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

from app.core.security import create_access_token, verify_password, hash_password, decode_token
from app.domain.services.auth_service import AuthService


@pytest.mark.asyncio
async def test_password_hashing():
    hashed = hash_password("testpass123")
    assert verify_password("testpass123", hashed) is True
    assert verify_password("wrongpass", hashed) is False


@pytest.mark.asyncio
async def test_create_and_decode_access_token():
    user_id = str(uuid4())
    token = create_access_token({"sub": user_id})
    payload = decode_token(token)
    assert payload is not None
    assert payload["sub"] == user_id
    assert payload["type"] == "access"


@pytest.mark.asyncio
async def test_decode_invalid_token():
    payload = decode_token("invalid.token.here")
    assert payload is None


@pytest.mark.asyncio
async def test_auth_login_success():
    user_id = uuid4()
    mock_user_repo = AsyncMock()
    from app.domain.entities.user import User
    from app.core.constants import UserRole
    mock_user_repo.get_by_email = AsyncMock(return_value=User(
        id=user_id,
        email="admin@test.com",
        hashed_password=hash_password("password123"),
        role=UserRole.ADMIN,
        is_active=True,
    ))

    mock_session_repo = AsyncMock()
    mock_session_repo.create = AsyncMock()

    service = AuthService(mock_user_repo, mock_session_repo)
    result = await service.login("admin@test.com", "password123")

    assert "access_token" in result
    assert "refresh_token" in result
    assert result["token_type"] == "bearer"


@pytest.mark.asyncio
async def test_auth_login_wrong_password():
    user_id = uuid4()
    mock_user_repo = AsyncMock()
    from app.domain.entities.user import User
    from app.core.constants import UserRole
    mock_user_repo.get_by_email = AsyncMock(return_value=User(
        id=user_id,
        email="admin@test.com",
        hashed_password=hash_password("password123"),
        role=UserRole.ADMIN,
        is_active=True,
    ))

    mock_session_repo = AsyncMock()
    service = AuthService(mock_user_repo, mock_session_repo)

    with pytest.raises(ValueError, match="Invalid email or password"):
        await service.login("admin@test.com", "wrongpassword")


@pytest.mark.asyncio
async def test_auth_login_user_not_found():
    mock_user_repo = AsyncMock()
    mock_user_repo.get_by_email = AsyncMock(return_value=None)

    mock_session_repo = AsyncMock()
    service = AuthService(mock_user_repo, mock_session_repo)

    with pytest.raises(ValueError, match="Invalid email or password"):
        await service.login("nobody@test.com", "password123")


@pytest.mark.asyncio
async def test_auth_login_inactive_user():
    user_id = uuid4()
    mock_user_repo = AsyncMock()
    from app.domain.entities.user import User
    from app.core.constants import UserRole
    mock_user_repo.get_by_email = AsyncMock(return_value=User(
        id=user_id,
        email="admin@test.com",
        hashed_password=hash_password("password123"),
        role=UserRole.ADMIN,
        is_active=False,
    ))

    mock_session_repo = AsyncMock()
    service = AuthService(mock_user_repo, mock_session_repo)

    with pytest.raises(ValueError, match="Account is disabled"):
        await service.login("admin@test.com", "password123")
