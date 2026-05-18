import hashlib
import secrets
from datetime import datetime, timedelta, timezone
from uuid import UUID

from fastapi import HTTPException, status

from app.core.logging import get_logger
from app.domain.entities.subscription import ApiKey
from app.repositories.billing_repo import SqlBillingRepository

logger = get_logger("services.api_key")

KEY_PREFIX = "qp_"


def _generate_key() -> tuple[str, str, str]:
    """Return (raw_key, key_prefix, key_hash)."""
    raw = KEY_PREFIX + secrets.token_urlsafe(32)
    prefix = raw[:12]
    key_hash = hashlib.sha256(raw.encode()).hexdigest()
    return raw, prefix, key_hash


class ApiKeyService:
    def __init__(self, billing_repo: SqlBillingRepository) -> None:
        self._repo = billing_repo

    async def create_key(
        self,
        org_id: UUID,
        user_id: UUID,
        name: str,
        scopes: list[str],
        expires_in_days: int | None,
    ) -> tuple[ApiKey, str]:
        """Returns (ApiKey entity, raw_key). Store raw_key once — it is not recoverable."""
        raw, prefix, key_hash = _generate_key()
        expires_at = None
        if expires_in_days:
            expires_at = datetime.now(timezone.utc) + timedelta(days=expires_in_days)

        key = await self._repo.create_api_key(
            ApiKey(
                org_id=org_id,
                user_id=user_id,
                name=name,
                key_hash=key_hash,
                key_prefix=prefix,
                scopes=scopes,
                expires_at=expires_at,
            )
        )
        logger.info("api_key_created", org_id=str(org_id), user_id=str(user_id), name=name)
        return key, raw

    async def validate_key(self, raw_key: str) -> ApiKey:
        key_hash = hashlib.sha256(raw_key.encode()).hexdigest()
        key = await self._repo.get_api_key_by_hash(key_hash)
        if not key:
            raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid API key")
        if key.expires_at and key.expires_at < datetime.now(timezone.utc):
            raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="API key expired")
        await self._repo.touch_api_key(key.id, datetime.now(timezone.utc))
        return key

    async def list_keys(self, org_id: UUID, user_id: UUID) -> list[ApiKey]:
        return await self._repo.list_api_keys(org_id, user_id)

    async def revoke_key(self, key_id: UUID, user_id: UUID, org_id: UUID) -> None:
        keys = await self._repo.list_api_keys(org_id, user_id)
        if not any(k.id == key_id for k in keys):
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="API key not found")
        await self._repo.revoke_api_key(key_id)
        logger.info("api_key_revoked", key_id=str(key_id), user_id=str(user_id))
