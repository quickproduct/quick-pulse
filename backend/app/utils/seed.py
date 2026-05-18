import asyncio

from sqlalchemy import select

from app.core.config import get_settings
from app.core.security import hash_password
from app.infrastructure.db.database import get_engine, get_session_factory
from app.infrastructure.db.models import PlanModel, UserModel


async def seed_admin():
    settings = get_settings()
    factory = get_session_factory()

    async with factory() as session:
        result = await session.execute(
            select(UserModel).where(UserModel.email == settings.DEFAULT_ADMIN_EMAIL)
        )
        existing = result.scalar_one_or_none()
        if existing:
            print(f"Admin user already exists: {settings.DEFAULT_ADMIN_EMAIL}")
        else:
            user = UserModel(
                email=settings.DEFAULT_ADMIN_EMAIL,
                hashed_password=hash_password(settings.DEFAULT_ADMIN_PASSWORD),
                # Plain string — no enum cast, avoids ::userrole error
                role="admin",
                is_active=True,
            )
            session.add(user)
            await session.commit()
            print(f"Created admin user: {settings.DEFAULT_ADMIN_EMAIL}")
            print(f"Password: {settings.DEFAULT_ADMIN_PASSWORD}")
            print("Change this password after first login!")


async def seed_plans():
    """Ensure default plans exist. Safe to run multiple times."""
    import uuid
    from datetime import datetime, timezone

    factory = get_session_factory()

    default_plans = [
        {
            "name": "free",
            "display_name": "Free",
            "max_hosts": 1,
            "max_users": 2,
            "max_containers": 10,
            "metrics_retention_days": 7,
            "price_monthly_cents": 0,
            "price_yearly_cents": 0,
            "features": {"alerts": True, "api_keys": False, "multi_host": False},
        },
        {
            "name": "pro",
            "display_name": "Pro",
            "max_hosts": 5,
            "max_users": 10,
            "max_containers": -1,
            "metrics_retention_days": 30,
            "price_monthly_cents": 1900,
            "price_yearly_cents": 19900,
            "features": {"alerts": True, "api_keys": True, "multi_host": True},
        },
        {
            "name": "enterprise",
            "display_name": "Enterprise",
            "max_hosts": -1,
            "max_users": -1,
            "max_containers": -1,
            "metrics_retention_days": 365,
            "price_monthly_cents": 9900,
            "price_yearly_cents": 99900,
            "features": {
                "alerts": True,
                "api_keys": True,
                "multi_host": True,
                "sso": True,
                "priority_support": True,
            },
        },
    ]

    async with get_session_factory()() as session:
        for plan_data in default_plans:
            result = await session.execute(
                select(PlanModel).where(PlanModel.name == plan_data["name"])
            )
            if result.scalar_one_or_none():
                print(f"Plan '{plan_data['name']}' already exists, skipping.")
                continue

            plan = PlanModel(**plan_data)
            session.add(plan)
            print(f"Created plan: {plan_data['display_name']}")

        await session.commit()


async def main():
    engine = get_engine()
    try:
        await seed_admin()
        try:
            await seed_plans()
        except Exception as exc:
            print(f"Warning: could not seed plans (migration 003 may not have run yet): {exc}")
    finally:
        await engine.dispose()


if __name__ == "__main__":
    asyncio.run(main())
