from uuid import UUID

from sqlalchemy import select, delete
from sqlalchemy.ext.asyncio import AsyncSession

from app.domain.entities.organization import Organization, OrgMember, OrgInvitation
from app.infrastructure.db.models import OrganizationModel, OrgMemberModel, OrgInvitationModel


def _org_from_model(m: OrganizationModel) -> Organization:
    return Organization(
        id=m.id,
        name=m.name,
        slug=m.slug,
        plan_id=m.plan_id,
        created_at=m.created_at,
        updated_at=m.updated_at,
    )


def _member_from_model(m: OrgMemberModel) -> OrgMember:
    return OrgMember(
        id=m.id,
        org_id=m.org_id,
        user_id=m.user_id,
        role=m.role,
        invited_by=m.invited_by,
        joined_at=m.joined_at,
    )


def _invitation_from_model(m: OrgInvitationModel) -> OrgInvitation:
    return OrgInvitation(
        id=m.id,
        org_id=m.org_id,
        email=m.email,
        role=m.role,
        token=m.token,
        invited_by=m.invited_by,
        expires_at=m.expires_at,
        accepted_at=m.accepted_at,
        created_at=m.created_at,
    )


class SqlOrganizationRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def create(self, org: Organization) -> Organization:
        model = OrganizationModel(
            name=org.name,
            slug=org.slug,
            plan_id=org.plan_id,
        )
        self._session.add(model)
        await self._session.flush()
        return _org_from_model(model)

    async def get_by_id(self, org_id: UUID) -> Organization | None:
        result = await self._session.execute(
            select(OrganizationModel).where(OrganizationModel.id == org_id)
        )
        model = result.scalar_one_or_none()
        return _org_from_model(model) if model else None

    async def get_by_slug(self, slug: str) -> Organization | None:
        result = await self._session.execute(
            select(OrganizationModel).where(OrganizationModel.slug == slug)
        )
        model = result.scalar_one_or_none()
        return _org_from_model(model) if model else None

    async def update(self, org: Organization) -> Organization:
        result = await self._session.execute(
            select(OrganizationModel).where(OrganizationModel.id == org.id)
        )
        model = result.scalar_one_or_none()
        if model is None:
            raise ValueError(f"Organization {org.id} not found")
        if org.name is not None:
            model.name = org.name
        if org.slug is not None:
            model.slug = org.slug
        await self._session.flush()
        return _org_from_model(model)

    # ── Members ──────────────────────────────────────────────────────────────

    async def add_member(self, member: OrgMember) -> OrgMember:
        model = OrgMemberModel(
            org_id=member.org_id,
            user_id=member.user_id,
            role=member.role,
            invited_by=member.invited_by,
        )
        self._session.add(model)
        await self._session.flush()
        return _member_from_model(model)

    async def get_member(self, org_id: UUID, user_id: UUID) -> OrgMember | None:
        result = await self._session.execute(
            select(OrgMemberModel).where(
                OrgMemberModel.org_id == org_id,
                OrgMemberModel.user_id == user_id,
            )
        )
        model = result.scalar_one_or_none()
        return _member_from_model(model) if model else None

    async def list_members(self, org_id: UUID) -> list[OrgMember]:
        result = await self._session.execute(
            select(OrgMemberModel).where(OrgMemberModel.org_id == org_id)
        )
        return [_member_from_model(m) for m in result.scalars().all()]

    async def remove_member(self, org_id: UUID, user_id: UUID) -> None:
        await self._session.execute(
            delete(OrgMemberModel).where(
                OrgMemberModel.org_id == org_id,
                OrgMemberModel.user_id == user_id,
            )
        )

    async def get_orgs_for_user(self, user_id: UUID) -> list[Organization]:
        result = await self._session.execute(
            select(OrganizationModel)
            .join(OrgMemberModel, OrgMemberModel.org_id == OrganizationModel.id)
            .where(OrgMemberModel.user_id == user_id)
        )
        return [_org_from_model(m) for m in result.scalars().all()]

    # ── Invitations ──────────────────────────────────────────────────────────

    async def create_invitation(self, inv: OrgInvitation) -> OrgInvitation:
        model = OrgInvitationModel(
            org_id=inv.org_id,
            email=inv.email,
            role=inv.role,
            token=inv.token,
            invited_by=inv.invited_by,
            expires_at=inv.expires_at,
        )
        self._session.add(model)
        await self._session.flush()
        return _invitation_from_model(model)

    async def get_invitation_by_token(self, token: str) -> OrgInvitation | None:
        result = await self._session.execute(
            select(OrgInvitationModel).where(OrgInvitationModel.token == token)
        )
        model = result.scalar_one_or_none()
        return _invitation_from_model(model) if model else None

    async def accept_invitation(self, invitation_id: UUID, accepted_at) -> None:
        result = await self._session.execute(
            select(OrgInvitationModel).where(OrgInvitationModel.id == invitation_id)
        )
        model = result.scalar_one_or_none()
        if model is None:
            raise ValueError(f"Invitation {invitation_id} not found")
        model.accepted_at = accepted_at
        await self._session.flush()
