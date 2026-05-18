from collections.abc import AsyncGenerator
from uuid import UUID

import redis.asyncio as aioredis
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import get_settings
from app.core.constants import UserRole
from app.core.security import decode_token
from app.domain.entities.user import User
from app.domain.interfaces.repositories import (
    AlertRepository,
    AlertRuleRepository,
    AuditLogRepository,
    ContainerEventRepository,
    ContainerRepository,
    HostRepository,
    MetricsRepository,
    SessionRepository,
    StackRepository,
    UserRepository,
)
from app.infrastructure.cache.redis import get_redis
from app.infrastructure.db.database import get_db_session
from app.infrastructure.docker.client import get_docker_client
from app.infrastructure.websocket.manager import ws_manager
from app.repositories.alert_repo import SqlAlertRepository, SqlAlertRuleRepository, SqlSessionRepository
from app.repositories.audit_log_repo import SqlAuditLogRepository
from app.repositories.billing_repo import SqlBillingRepository
from app.repositories.container_event_repo import SqlContainerEventRepository
from app.repositories.container_repo import SqlContainerRepository
from app.repositories.host_repo import SqlHostRepository
from app.repositories.metrics_repo import SqlMetricsRepository
from app.repositories.organization_repo import SqlOrganizationRepository
from app.repositories.stack_repo import SqlStackRepository
from app.repositories.user_repo import SqlUserRepository

_bearer_scheme = HTTPBearer()


async def get_session() -> AsyncGenerator[AsyncSession, None]:
    async for session in get_db_session():
        yield session


def get_user_repo(session: AsyncSession = Depends(get_session)) -> UserRepository:
    return SqlUserRepository(session)


def get_host_repo(session: AsyncSession = Depends(get_session)) -> HostRepository:
    return SqlHostRepository(session)


def get_metrics_repo(session: AsyncSession = Depends(get_session)) -> MetricsRepository:
    return SqlMetricsRepository(session)


def get_container_repo(session: AsyncSession = Depends(get_session)) -> ContainerRepository:
    return SqlContainerRepository(session)


def get_container_event_repo(session: AsyncSession = Depends(get_session)) -> ContainerEventRepository:
    return SqlContainerEventRepository(session)


def get_stack_repo(session: AsyncSession = Depends(get_session)) -> StackRepository:
    return SqlStackRepository(session)


def get_alert_rule_repo(session: AsyncSession = Depends(get_session)) -> AlertRuleRepository:
    return SqlAlertRuleRepository(session)


def get_alert_repo(session: AsyncSession = Depends(get_session)) -> AlertRepository:
    return SqlAlertRepository(session)


def get_audit_repo(session: AsyncSession = Depends(get_session)) -> AuditLogRepository:
    return SqlAuditLogRepository(session)


def get_session_repo(session: AsyncSession = Depends(get_session)) -> SessionRepository:
    return SqlSessionRepository(session)


# NOTE: Use get_redis_dep() (async) in FastAPI dependencies; the old sync
# get_redis_client() has been removed — it called run_until_complete() inside
# a running event loop which always raises RuntimeError.


async def get_redis_dep() -> aioredis.Redis:
    return await get_redis()


async def get_docker():
    return await get_docker_client()


def get_ws_manager():
    return ws_manager


async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(_bearer_scheme),
    user_repo: UserRepository = Depends(get_user_repo),
) -> User:
    payload = decode_token(credentials.credentials)
    if payload is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid or expired token")
    if payload.get("type") != "access":
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid token type")
    user_id = payload.get("sub")
    if user_id is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid token payload")
    user = await user_repo.get_by_id(UUID(user_id))
    if user is None or not user.is_active:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="User not found or inactive")
    return user


async def require_admin(current_user: User = Depends(get_current_user)) -> User:
    # Coerce enum or plain string role value for safe comparison
    role_val = current_user.role.value if hasattr(current_user.role, "value") else current_user.role
    if role_val != UserRole.ADMIN.value:
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Admin access required")
    return current_user


def get_org_repo(session: AsyncSession = Depends(get_session)) -> SqlOrganizationRepository:
    return SqlOrganizationRepository(session)


def get_billing_repo(session: AsyncSession = Depends(get_session)) -> SqlBillingRepository:
    return SqlBillingRepository(session)
