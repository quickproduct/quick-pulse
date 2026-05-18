from uuid import uuid4

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.constants import EventType
from app.domain.entities.event import ContainerEvent
from app.domain.interfaces.repositories import ContainerEventRepository
from app.infrastructure.db.models import ContainerEventModel


def _model_to_entity(model: ContainerEventModel) -> ContainerEvent:
    return ContainerEvent(
        id=model.id,
        container_docker_id=model.container_docker_id,
        container_name=model.container_name,
        event_type=model.event_type,
        timestamp=model.timestamp,
        metadata=model.metadata_,
    )


class SqlContainerEventRepository(ContainerEventRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def insert(self, event: ContainerEvent) -> ContainerEvent:
        model = ContainerEventModel(
            container_docker_id=event.container_docker_id,
            container_name=event.container_name,
            event_type=event.event_type,
            timestamp=event.timestamp,
            metadata_=event.metadata,
        )
        self._session.add(model)
        await self._session.flush()
        return _model_to_entity(model)

    async def get_recent(self, limit: int = 50) -> list[ContainerEvent]:
        result = await self._session.execute(
            select(ContainerEventModel).order_by(ContainerEventModel.timestamp.desc()).limit(limit)
        )
        return [_model_to_entity(m) for m in result.scalars().all()]
