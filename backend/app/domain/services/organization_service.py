import secrets
from datetime import datetime, timedelta, timezone
from uuid import UUID

from fastapi import HTTPException, status

from app.core.logging import get_logger
from app.domain.entities.organization import OrgInvitation, OrgMember, Organization
from app.repositories.organization_repo import SqlOrganizationRepository

logger = get_logger("services.organization")

INVITE_EXPIRY_HOURS = 48


class OrganizationService:
    def __init__(self, org_repo: SqlOrganizationRepository) -> None:
        self._repo = org_repo

    async def create_organization(self, name: str, slug: str, owner_id: UUID) -> Organization:
        existing = await self._repo.get_by_slug(slug)
        if existing:
            raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail="Slug already taken")

        org = await self._repo.create(Organization(name=name, slug=slug))
        await self._repo.add_member(
            OrgMember(org_id=org.id, user_id=owner_id, role="owner")
        )
        logger.info("organization_created", org_id=str(org.id), slug=slug, owner=str(owner_id))
        return org

    async def get_org_for_user(self, user_id: UUID) -> Organization | None:
        orgs = await self._repo.get_orgs_for_user(user_id)
        return orgs[0] if orgs else None

    async def get_org_by_id(self, org_id: UUID) -> Organization:
        org = await self._repo.get_by_id(org_id)
        if not org:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Organization not found")
        return org

    async def update_org(self, org_id: UUID, name: str | None, slug: str | None) -> Organization:
        org = await self.get_org_by_id(org_id)
        if slug and slug != org.slug:
            existing = await self._repo.get_by_slug(slug)
            if existing:
                raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail="Slug already taken")
        org.name = name or org.name
        org.slug = slug or org.slug
        updated = await self._repo.update(org)
        logger.info("organization_updated", org_id=str(org_id))
        return updated

    async def invite_member(self, org_id: UUID, email: str, role: str, invited_by: UUID) -> OrgInvitation:
        token = secrets.token_hex(32)
        invitation = await self._repo.create_invitation(
            OrgInvitation(
                org_id=org_id,
                email=email,
                role=role,
                token=token,
                invited_by=invited_by,
                expires_at=datetime.now(timezone.utc) + timedelta(hours=INVITE_EXPIRY_HOURS),
            )
        )
        logger.info("invitation_created", org_id=str(org_id), email=email, role=role)
        return invitation

    async def accept_invitation(self, token: str, user_id: UUID) -> OrgMember:
        inv = await self._repo.get_invitation_by_token(token)
        if not inv:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Invitation not found or expired")
        if inv.accepted_at is not None:
            raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail="Invitation already accepted")
        if inv.expires_at < datetime.now(timezone.utc):
            raise HTTPException(status_code=status.HTTP_410_GONE, detail="Invitation expired")

        existing = await self._repo.get_member(inv.org_id, user_id)
        if existing:
            raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail="Already a member of this organization")

        member = await self._repo.add_member(
            OrgMember(org_id=inv.org_id, user_id=user_id, role=inv.role, invited_by=inv.invited_by)
        )
        await self._repo.accept_invitation(inv.id, datetime.now(timezone.utc))
        logger.info("invitation_accepted", org_id=str(inv.org_id), user_id=str(user_id))
        return member

    async def list_members(self, org_id: UUID) -> list[OrgMember]:
        return await self._repo.list_members(org_id)

    async def remove_member(self, org_id: UUID, user_id: UUID, requester_id: UUID) -> None:
        requester = await self._repo.get_member(org_id, requester_id)
        if not requester or requester.role != "owner":
            raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Only the owner can remove members")
        if user_id == requester_id:
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Owner cannot remove themselves")
        await self._repo.remove_member(org_id, user_id)
        logger.info("member_removed", org_id=str(org_id), user_id=str(user_id))

    async def require_membership(self, org_id: UUID, user_id: UUID, min_role: str = "viewer") -> OrgMember:
        """Raise 403 if the user is not a member with at least min_role."""
        role_rank = {"viewer": 0, "member": 1, "admin": 2, "owner": 3}
        member = await self._repo.get_member(org_id, user_id)
        if not member:
            raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Not a member of this organization")
        if role_rank.get(member.role, -1) < role_rank.get(min_role, 0):
            raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail=f"Requires at least {min_role} role")
        return member
