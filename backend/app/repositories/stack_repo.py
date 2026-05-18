from uuid import UUID

from sqlalchemy import select, delete
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.constants import ContainerStatus, StackStatus
from app.domain.entities.stack import ComposeService, ComposeStack
from app.domain.interfaces.repositories import StackRepository
from app.infrastructure.db.models import ComposeServiceModel, ComposeStackModel


def _stack_model_to_entity(model: ComposeStackModel) -> ComposeStack:
    return ComposeStack(
        id=model.id,
        name=model.name,
        project_dir=model.project_dir,
        status=model.status,
        services_count=model.services_count,
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


class SqlStackRepository(StackRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def get_by_name(self, name: str) -> ComposeStack | None:
        result = await self._session.execute(select(ComposeStackModel).where(ComposeStackModel.name == name))
        model = result.scalar_one_or_none()
        return _stack_model_to_entity(model) if model else None

    async def list_all(self) -> list[ComposeStack]:
        result = await self._session.execute(select(ComposeStackModel).order_by(ComposeStackModel.name))
        return [_stack_model_to_entity(m) for m in result.scalars().all()]

    async def upsert(self, stack: ComposeStack) -> ComposeStack:
        existing = await self.get_by_name(stack.name)
        if existing:
            from sqlalchemy import update as sa_update
            await self._session.execute(
                sa_update(ComposeStackModel)
                .where(ComposeStackModel.id == existing.id)
                .values(
                    status=stack.status,
                    services_count=stack.services_count,
                    project_dir=stack.project_dir,
                )
            )
            await self._session.flush()
            return await self.get_by_name(stack.name) or stack
        model = ComposeStackModel(
            name=stack.name,
            project_dir=stack.project_dir,
            status=stack.status,
            services_count=stack.services_count,
        )
        self._session.add(model)
        await self._session.flush()
        return _stack_model_to_entity(model)

    async def upsert_services(self, stack_id: UUID, services: list[ComposeService]) -> list[ComposeService]:
        async with self._session.begin_nested():
            await self._session.execute(
                delete(ComposeServiceModel).where(ComposeServiceModel.stack_id == stack_id)
            )
            entities = []
            for svc in services:
                model = ComposeServiceModel(
                    stack_id=stack_id,
                    name=svc.name,
                    status=svc.status,
                    ports=svc.ports,
                )
                self._session.add(model)
                entities.append(svc)
            await self._session.flush()
        return entities

    async def delete(self, name: str) -> None:
        stack = await self.get_by_name(name)
        if stack:
            await self._session.execute(delete(ComposeServiceModel).where(ComposeServiceModel.stack_id == stack.id))
            await self._session.execute(delete(ComposeStackModel).where(ComposeStackModel.id == stack.id))
