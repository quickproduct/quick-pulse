from dataclasses import dataclass, field
from datetime import datetime
from uuid import UUID


@dataclass
class Organization:
    name: str
    slug: str
    id: UUID | None = None
    plan_id: UUID | None = None
    created_at: datetime | None = None
    updated_at: datetime | None = None


@dataclass
class OrgMember:
    org_id: UUID
    user_id: UUID
    role: str = "member"
    id: UUID | None = None
    invited_by: UUID | None = None
    joined_at: datetime | None = None


@dataclass
class OrgInvitation:
    org_id: UUID
    email: str
    role: str
    token: str
    invited_by: UUID
    expires_at: datetime
    id: UUID | None = None
    accepted_at: datetime | None = None
    created_at: datetime | None = None
