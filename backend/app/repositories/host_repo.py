from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.domain.entities.host import Host, HostMetric
from app.domain.interfaces.repositories import HostRepository
from app.infrastructure.db.models import HostModel


def _model_to_entity(model: HostModel) -> Host:
    return Host(
        id=model.id,
        hostname=model.hostname,
        ip_address=model.ip_address,
        os_info=model.os_info,
        cpu_count=model.cpu_count,
        total_memory=model.total_memory,
        total_disk=model.total_disk,
        created_at=model.created_at,
    )


class SqlHostRepository(HostRepository):
    def __init__(self, session: AsyncSession):
        self._session = session

    async def get_by_hostname(self, hostname: str) -> Host | None:
        result = await self._session.execute(select(HostModel).where(HostModel.hostname == hostname))
        model = result.scalar_one_or_none()
        return _model_to_entity(model) if model else None

    async def get_current(self) -> Host | None:
        import socket
        hostname = socket.gethostname()
        return await self.get_by_hostname(hostname)

    async def upsert(self, host: Host) -> Host:
        existing = await self.get_by_hostname(host.hostname)
        if existing:
            return existing
        model = HostModel(
            hostname=host.hostname,
            ip_address=host.ip_address,
            os_info=host.os_info,
            cpu_count=host.cpu_count,
            total_memory=host.total_memory,
            total_disk=host.total_disk,
        )
        self._session.add(model)
        await self._session.flush()
        return _model_to_entity(model)
