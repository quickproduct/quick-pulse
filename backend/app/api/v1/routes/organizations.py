from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, status

from app.core.logging import get_logger
from app.domain.entities.user import User
from app.domain.services.organization_service import OrganizationService
from app.schemas.organization import (
    AcceptInviteRequest,
    InviteRequest,
    InvitationResponse,
    MemberResponse,
    OrgCreate,
    OrgResponse,
    OrgUpdate,
)
from app.utils.deps import get_billing_repo, get_current_user, get_org_repo, require_admin

router = APIRouter()
logger = get_logger("api.organizations")


def _get_org_service(org_repo=Depends(get_org_repo)) -> OrganizationService:
    return OrganizationService(org_repo)


@router.post("", response_model=OrgResponse, status_code=201)
async def create_organization(
    body: OrgCreate,
    current_user: User = Depends(require_admin),
    svc: OrganizationService = Depends(_get_org_service),
    billing_repo=Depends(get_billing_repo),
):
    """Create a new organization. The calling user becomes owner."""
    from app.domain.services.billing_service import BillingService
    from app.core.config import get_settings

    logger.info("org_create", name=body.name, user_id=str(current_user.id))
    try:
        org = await svc.create_organization(body.name, body.slug, current_user.id)

        settings = get_settings()
        billing_svc = BillingService(billing_repo, trial_days=settings.TRIAL_DAYS)
        try:
            await billing_svc.create_trial_subscription(org.id, settings.DEFAULT_PLAN)
        except Exception as e:
            logger.error("org_billing_trial_failed", org_id=str(org.id), error=str(e), exc_info=True)
            # Non-fatal: org exists but without a trial subscription

        logger.info("organization_created_with_trial", org_id=str(org.id))
        return OrgResponse(
            id=org.id,
            name=org.name,
            slug=org.slug,
            plan_id=org.plan_id,
            created_at=org.created_at,
            updated_at=org.updated_at,
        )
    except HTTPException:
        raise
    except Exception as e:
        logger.error("org_create_error", name=body.name, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to create organization")


@router.get("/me", response_model=OrgResponse | None)
async def get_my_org(
    current_user: User = Depends(get_current_user),
    svc: OrganizationService = Depends(_get_org_service),
):
    """Return the first organization the current user belongs to."""
    try:
        return await svc.get_org_for_user(current_user.id)
    except Exception as e:
        logger.error("org_get_me_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to get organization")


@router.put("/me", response_model=OrgResponse)
async def update_my_org(
    body: OrgUpdate,
    current_user: User = Depends(require_admin),
    svc: OrganizationService = Depends(_get_org_service),
):
    try:
        org = await svc.get_org_for_user(current_user.id)
        if not org:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="No organization found")
        result = await svc.update_org(org.id, body.name, body.slug)
        logger.info("org_updated", org_id=str(org.id), user_id=str(current_user.id))
        return result
    except HTTPException:
        raise
    except Exception as e:
        logger.error("org_update_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to update organization")


@router.get("/members", response_model=list[MemberResponse])
async def list_members(
    current_user: User = Depends(require_admin),
    svc: OrganizationService = Depends(_get_org_service),
):
    try:
        org = await svc.get_org_for_user(current_user.id)
        if not org:
            raise HTTPException(status_code=404, detail="No organization found")
        return await svc.list_members(org.id)
    except HTTPException:
        raise
    except Exception as e:
        logger.error("org_list_members_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to list members")


@router.post("/members/invite", response_model=InvitationResponse, status_code=201)
async def invite_member(
    body: InviteRequest,
    current_user: User = Depends(require_admin),
    svc: OrganizationService = Depends(_get_org_service),
):
    try:
        org = await svc.get_org_for_user(current_user.id)
        if not org:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="No organization found")
        invitation = await svc.invite_member(org.id, body.email, body.role, current_user.id)
        logger.info("member_invited", org_id=str(org.id), email=body.email)
        return invitation
    except HTTPException:
        raise
    except Exception as e:
        logger.error("org_invite_member_error", user_id=str(current_user.id), email=body.email, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to invite member")


@router.delete("/members/{user_id}", status_code=204)
async def remove_member(
    user_id: UUID,
    current_user: User = Depends(require_admin),
    svc: OrganizationService = Depends(_get_org_service),
):
    try:
        org = await svc.get_org_for_user(current_user.id)
        if not org:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="No organization found")
        await svc.remove_member(org.id, user_id, current_user.id)
        logger.info("member_removed", org_id=str(org.id), removed_user=str(user_id), by=str(current_user.id))
    except HTTPException:
        raise
    except Exception as e:
        logger.error("org_remove_member_error", user_id=str(user_id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to remove member")


@router.post("/invitations/accept", response_model=MemberResponse)
async def accept_invitation(
    body: AcceptInviteRequest,
    current_user: User = Depends(get_current_user),
    svc: OrganizationService = Depends(_get_org_service),
):
    try:
        member = await svc.accept_invitation(body.token, current_user.id)
        logger.info("invitation_accepted_via_api", user_id=str(current_user.id))
        return member
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error("org_accept_invite_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to accept invitation")
