from datetime import datetime, timedelta, timezone
from uuid import UUID

from app.core.logging import get_logger
from app.core.security import (
    create_access_token,
    create_refresh_token,
    decode_token,
    hash_password,
    verify_password,
)
from app.domain.entities.user import User
from app.domain.interfaces.repositories import SessionRepository, UserRepository

logger = get_logger("auth_service")


class AuthService:
    def __init__(self, user_repo: UserRepository, session_repo: SessionRepository):
        self._user_repo = user_repo
        self._session_repo = session_repo

    async def login(self, email: str, password: str) -> dict:
        user = await self._user_repo.get_by_email(email)
        if not user or not verify_password(password, user.hashed_password):
            logger.warning("login_failed", email=email)
            raise ValueError("Invalid email or password")
        if not user.is_active:
            raise ValueError("Account is disabled")

        access_token = create_access_token({"sub": str(user.id)})
        refresh_token = create_refresh_token({"sub": str(user.id)})

        from app.core.config import get_settings
        settings = get_settings()
        expires_at = datetime.now(timezone.utc) + timedelta(days=settings.REFRESH_TOKEN_EXPIRE_DAYS)
        await self._session_repo.create(user.id, refresh_token, expires_at)

        logger.info("login_success", user_id=str(user.id))
        return {
            "access_token": access_token,
            "refresh_token": refresh_token,
            "token_type": "bearer",
            "user": user,
        }

    async def refresh(self, refresh_token: str) -> dict:
        payload = decode_token(refresh_token)
        if not payload or payload.get("type") != "refresh":
            raise ValueError("Invalid refresh token")

        session = await self._session_repo.get_by_refresh_token(refresh_token)
        if not session:
            raise ValueError("Session not found")

        user = await self._user_repo.get_by_id(session["user_id"])
        if not user or not user.is_active:
            raise ValueError("User not found or inactive")

        await self._session_repo.delete_by_refresh_token(refresh_token)

        new_access = create_access_token({"sub": str(user.id)})
        new_refresh = create_refresh_token({"sub": str(user.id)})

        from app.core.config import get_settings
        settings = get_settings()
        expires_at = datetime.now(timezone.utc) + timedelta(days=settings.REFRESH_TOKEN_EXPIRE_DAYS)
        await self._session_repo.create(user.id, new_refresh, expires_at)

        return {"access_token": new_access, "refresh_token": new_refresh, "token_type": "bearer"}

    async def logout(self, refresh_token: str) -> bool:
        return await self._session_repo.delete_by_refresh_token(refresh_token)

    async def change_password(self, user_id: UUID, current_password: str, new_password: str) -> bool:
        user = await self._user_repo.get_by_id(user_id)
        if not user:
            raise ValueError("User not found")
        if not verify_password(current_password, user.hashed_password):
            raise ValueError("Current password is incorrect")
        await self._user_repo.update_password(user_id, hash_password(new_password))
        return True

    async def register(self, email: str, password: str, role: str = "admin") -> User:
        existing = await self._user_repo.get_by_email(email)
        if existing:
            raise ValueError("Email already registered")
        user = await self._user_repo.create(email, hash_password(password), role)
        logger.info("user_registered", user_id=str(user.id), email=email)
        return user
