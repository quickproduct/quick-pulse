from dataclasses import dataclass, field
from datetime import datetime
from uuid import UUID

from app.core.constants import UserRole


@dataclass
class User:
    id: UUID
    email: str
    hashed_password: str
    role: UserRole = UserRole.ADMIN
    is_active: bool = True
    created_at: datetime | None = None
    updated_at: datetime | None = None
