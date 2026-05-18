from datetime import datetime, timezone
from uuid import UUID, uuid4

from sqlalchemy import select, update, delete
from sqlalchemy.ext.asyncio import AsyncSession

from app.domain.entities.user import User
from app.domain.interfaces.repositories import UserRepository
from app.infrastructure.db.models import UserModel


def _model_to_entity(model: UserModel) -> User:
    return User(
        id=model.id,
        email=model.email,
        hashed_password=model.hashed_password,
        role=model.role,
        is_active=model.is_active,
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


class SqlUserRepository(UserRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def get_by_id(self, user_id: UUID) -> User | None:
        result = await self._session.execute(select(UserModel).where(UserModel.id == user_id))
        model = result.scalar_one_or_none()
        return _model_to_entity(model) if model else None

    async def get_by_email(self, email: str) -> User | None:
        result = await self._session.execute(select(UserModel).where(UserModel.email == email))
        model = result.scalar_one_or_none()
        return _model_to_entity(model) if model else None

    async def create(self, email: str, hashed_password: str, role: str = "admin") -> User:
        model = UserModel(email=email, hashed_password=hashed_password, role=role)
        self._session.add(model)
        await self._session.flush()
        return _model_to_entity(model)

    async def update_password(self, user_id: UUID, hashed_password: str) -> None:
        await self._session.execute(
            update(UserModel).where(UserModel.id == user_id).values(hashed_password=hashed_password)
        )
