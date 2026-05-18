import time
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Request, status
from pydantic import BaseModel, EmailStr

from app.core.config import get_settings
from app.core.logging import get_logger
from app.domain.entities.user import User
from app.domain.services.auth_service import AuthService
from app.infrastructure.cache.redis import get_redis
from app.schemas.auth import LoginRequest, LoginResponse, RefreshRequest, ChangePasswordRequest, UserResponse
from app.schemas.billing import ApiKeyCreate, ApiKeyCreatedResponse, ApiKeyResponse
from app.utils.deps import (
    get_billing_repo,
    get_current_user,
    get_org_repo,
    get_session_repo,
    get_user_repo,
    require_admin,
)

router = APIRouter()
logger = get_logger("api.auth")


def _get_auth_service(session=Depends(get_session_repo), user=Depends(get_user_repo)):
    return AuthService(user, session)


class _LogoutBody(BaseModel):
    refresh_token: str


async def _check_login_rate(request: Request, redis=Depends(get_redis)):
    settings = get_settings()
    limit = settings.RATE_LIMIT_LOGIN_PER_MINUTE
    ip = request.client.host if request.client else "unknown"
    window = 60
    now = int(time.time())
    key = f"rl:login:{ip}:{now // window}"
    count = await redis.incr(key)
    if count == 1:
        await redis.expire(key, window * 2)
    if count > limit:
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail=f"Too many login attempts. Try again in {window - (now % window)} seconds.",
            headers={"Retry-After": str(window - (now % window))},
        )


@router.post("/login", response_model=LoginResponse)
async def login(
    request: Request,
    body: LoginRequest,
    auth_service: AuthService = Depends(_get_auth_service),
    _: None = Depends(_check_login_rate),
):
    logger.info("login_attempt", email=body.email)
    try:
        result = await auth_service.login(body.email, body.password)
        logger.info("login_success", email=body.email)
        return LoginResponse(
            access_token=result["access_token"],
            refresh_token=result["refresh_token"],
        )
    except ValueError as exc:
        logger.warning("login_failed", email=body.email, error=str(exc))
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid email or password")


@router.post("/logout")
async def logout(body: _LogoutBody, auth_service: AuthService = Depends(_get_auth_service)):
    await auth_service.logout(body.refresh_token)
    logger.info("logout_success")
    return {"message": "Logged out"}


@router.post("/refresh", response_model=LoginResponse)
async def refresh(body: RefreshRequest, auth_service: AuthService = Depends(_get_auth_service)):
    logger.info("token_refresh_attempt")
    try:
        result = await auth_service.refresh(body.refresh_token)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail=str(exc))
    logger.info("token_refresh_success")
    return LoginResponse(
        access_token=result["access_token"],
        refresh_token=result["refresh_token"],
    )


@router.get("/me", response_model=UserResponse)
async def me(current_user: User = Depends(get_current_user)):
    return UserResponse(
        id=current_user.id,
        email=current_user.email,
        role=current_user.role.value if hasattr(current_user.role, "value") else current_user.role,
        is_active=current_user.is_active,
        created_at=current_user.created_at,
    )


@router.put("/password")
async def change_password(
    body: ChangePasswordRequest,
    current_user: User = Depends(get_current_user),
    auth_service: AuthService = Depends(_get_auth_service),
):
    logger.info("password_change_attempt", user_id=str(current_user.id))
    try:
        await auth_service.change_password(current_user.id, body.current_password, body.new_password)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc))
    logger.info("password_change_success", user_id=str(current_user.id))
    return {"message": "Password changed"}


# ── Registration (when ALLOW_REGISTRATION=true) ───────────────────────────────

class RegisterRequest(BaseModel):
    email: EmailStr
    password: str


@router.post("/register", response_model=UserResponse, status_code=201)
async def register(
    body: RegisterRequest,
    auth_service: AuthService = Depends(_get_auth_service),
):
    settings = get_settings()
    if not settings.ALLOW_REGISTRATION:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Open registration is disabled. Contact an admin for an invite.",
        )
    logger.info("registration_attempt", email=body.email)
    try:
        user = await auth_service.register(body.email, body.password)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail=str(exc))
    logger.info("registration_success", email=body.email, user_id=str(user.id))
    return UserResponse(
        id=user.id,
        email=user.email,
        role=user.role.value if hasattr(user.role, "value") else user.role,
        is_active=user.is_active,
        created_at=user.created_at,
    )


# ── API Keys ──────────────────────────────────────────────────────────────────

@router.post("/api-keys", response_model=ApiKeyCreatedResponse, status_code=201)
async def create_api_key(
    body: ApiKeyCreate,
    current_user: User = Depends(get_current_user),
    org_repo=Depends(get_org_repo),
    billing_repo=Depends(get_billing_repo),
):
    from app.domain.services.api_key_service import ApiKeyService
    from app.domain.services.organization_service import OrganizationService

    org_svc = OrganizationService(org_repo)
    org = await org_svc.get_org_for_user(current_user.id)
    if not org:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="No organization found for this user")

    api_key_svc = ApiKeyService(billing_repo)
    key, raw_key = await api_key_svc.create_key(
        org_id=org.id,
        user_id=current_user.id,
        name=body.name,
        scopes=body.scopes,
        expires_in_days=body.expires_in_days,
    )
    logger.info("api_key_created_via_api", user_id=str(current_user.id), name=body.name)
    return ApiKeyCreatedResponse(
        id=key.id,
        name=key.name,
        key_prefix=key.key_prefix,
        scopes=key.scopes,
        is_active=key.is_active,
        last_used_at=key.last_used_at,
        expires_at=key.expires_at,
        created_at=key.created_at,
        raw_key=raw_key,
    )


@router.get("/api-keys", response_model=list[ApiKeyResponse])
async def list_api_keys(
    current_user: User = Depends(get_current_user),
    org_repo=Depends(get_org_repo),
    billing_repo=Depends(get_billing_repo),
):
    from app.domain.services.api_key_service import ApiKeyService
    from app.domain.services.organization_service import OrganizationService

    org_svc = OrganizationService(org_repo)
    org = await org_svc.get_org_for_user(current_user.id)
    if not org:
        return []

    api_key_svc = ApiKeyService(billing_repo)
    keys = await api_key_svc.list_keys(org.id, current_user.id)
    return [
        ApiKeyResponse(
            id=k.id,
            name=k.name,
            key_prefix=k.key_prefix,
            scopes=k.scopes,
            is_active=k.is_active,
            last_used_at=k.last_used_at,
            expires_at=k.expires_at,
            created_at=k.created_at,
        )
        for k in keys
    ]


@router.delete("/api-keys/{key_id}", status_code=204)
async def revoke_api_key(
    key_id: UUID,
    current_user: User = Depends(get_current_user),
    org_repo=Depends(get_org_repo),
    billing_repo=Depends(get_billing_repo),
):
    from app.domain.services.api_key_service import ApiKeyService
    from app.domain.services.organization_service import OrganizationService

    org_svc = OrganizationService(org_repo)
    org = await org_svc.get_org_for_user(current_user.id)
    if not org:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="No organization found")

    api_key_svc = ApiKeyService(billing_repo)
    await api_key_svc.revoke_key(key_id, current_user.id, org.id)
    logger.info("api_key_revoked_via_api", key_id=str(key_id), user_id=str(current_user.id))
