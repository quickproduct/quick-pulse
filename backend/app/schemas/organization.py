from datetime import datetime
from uuid import UUID

from pydantic import BaseModel, EmailStr, field_validator
import re


class OrgCreate(BaseModel):
    name: str
    slug: str

    @field_validator("slug")
    @classmethod
    def validate_slug(cls, v: str) -> str:
        if not re.match(r"^[a-z0-9-]+$", v):
            raise ValueError("slug must be lowercase alphanumeric with hyphens only")
        return v


class OrgUpdate(BaseModel):
    name: str | None = None
    slug: str | None = None


class OrgResponse(BaseModel):
    id: UUID
    name: str
    slug: str
    plan_id: UUID | None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class InviteRequest(BaseModel):
    email: EmailStr
    role: str = "member"

    @field_validator("role")
    @classmethod
    def validate_role(cls, v: str) -> str:
        allowed = {"owner", "admin", "member", "viewer"}
        if v not in allowed:
            raise ValueError(f"role must be one of {allowed}")
        return v


class AcceptInviteRequest(BaseModel):
    token: str


class MemberResponse(BaseModel):
    id: UUID
    org_id: UUID
    user_id: UUID
    role: str
    joined_at: datetime

    model_config = {"from_attributes": True}


class InvitationResponse(BaseModel):
    id: UUID
    org_id: UUID
    email: str
    role: str
    token: str
    expires_at: datetime
    accepted_at: datetime | None
    created_at: datetime

    model_config = {"from_attributes": True}
