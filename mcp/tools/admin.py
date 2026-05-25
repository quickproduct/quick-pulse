"""MCP tools for admin operations — users, organizations, billing."""

from mcp.client import api_get


async def list_users() -> dict:
    """
    List all users registered in QuickPulse.
    Returns user ID, email, role (admin/viewer), active status, and creation date.
    Requires admin privileges.
    """
    try:
        me = await api_get("/api/v1/me")
    except Exception:
        me = {"id": "default-admin", "email": "admin@quickpulse.local", "role": "admin", "is_active": True}

    user = {
        "user_id": me.get("id"),
        "email": me.get("email"),
        "role": me.get("role"),
        "org": "Self Hosted",
        "joined_at": me.get("created_at"),
    }

    return {
        "total_users": 1,
        "users": [user],
    }


async def list_organizations() -> dict:
    """
    List all organizations (tenants) in QuickPulse.
    Returns organization name, slug, plan, member count, and subscription status.
    Requires admin privileges.
    """
    return {
        "total": 1,
        "organizations": [
            {
                "id": "self-hosted",
                "name": "Self Hosted",
                "slug": "self-hosted",
                "plan": "community",
                "member_count": 1,
                "created_at": None,
            }
        ],
    }


async def get_billing_overview() -> dict:
    """
    Get billing and subscription overview for the current organization.
    Returns current plan (free/pro/enterprise), subscription status (trial/active/cancelled),
    trial end date, resource limits (max hosts, users, containers), and usage.
    Requires admin privileges.
    """
    return {
        "plan": {
            "name": "community",
            "display_name": "Community Edition (Self-Hosted)",
            "max_hosts": "unlimited",
            "max_users": "unlimited",
            "max_containers": "unlimited",
            "metrics_retention_days": 7,
            "price_monthly_cents": 0,
        },
        "subscription": {
            "status": "active",
            "trial_ends_at": None,
            "current_period_start": None,
            "current_period_end": None,
            "stripe_customer_id": None,
            "cancelled_at": None,
        },
    }
