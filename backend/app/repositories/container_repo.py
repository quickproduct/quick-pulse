from uuid import UUID

from sqlalchemy import select, delete
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.constants import ContainerStatus
from app.domain.entities.container import Container
from app.domain.interfaces.repositories import ContainerRepository
from app.infrastructure.db.models import ContainerModel


def _model_to_entity(model: ContainerModel) -> Container:
    return Container(
        id=model.id,
        docker_id=model.docker_id,
        name=model.name,
        image=model.image,
        status=model.status,
        ports=model.ports,
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


class SqlContainerRepository(ContainerRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def get_by_docker_id(self, docker_id: str) -> Container | None:
        result = await self._session.execute(select(ContainerModel).where(ContainerModel.docker_id == docker_id))
        model = result.scalar_one_or_none()
        return _model_to_entity(model) if model else None

    async def upsert(self, container: Container) -> Container:
        existing = await self.get_by_docker_id(container.docker_id)
        if existing:
            return existing
        model = ContainerModel(
            docker_id=container.docker_id,
            name=container.name,
            image=container.image,
            status=container.status,
            ports=container.ports,
        )
        self._session.add(model)
        await self._session.flush()
        return _model_to_entity(model)

    async def upsert_batch(self, containers: list[Container]) -> list[Container]:
        result = []
        for c in containers:
            result.append(await self.upsert(c))
        return result

    async def remove_stale(self, active_docker_ids: list[str]) -> int:
        if not active_docker_ids:
            stmt = delete(ContainerModel)
        else:
            stmt = delete(ContainerModel).where(ContainerModel.docker_id.notin_(active_docker_ids))
        r = await self._session.execute(stmt)
        return r.rowcount
