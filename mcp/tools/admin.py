"""MCP tools for admin operations — users, organizations, billing."""

from mcp.client import api_get


async def list_users() -> dict:
    """
    List all users registered in QuickPulse.
    Returns user ID, email, role (admin/viewer), active status, and creation date.
    Requires admin privileges.
    """
    data = await api_get("/api/v1/organizations")
    # Users are accessed via the organizations endpoint
    orgs = data if isinstance(data, list) else []
    all_members = []
    for org in orgs:
        members = org.get("members") or []
        for m in members:
            all_members.append({
                "user_id": m.get("user_id"),
                "email": m.get("email"),
                "role": m.get("role"),
                "org": org.get("name"),
                "joined_at": m.get("joined_at"),
            })

    return {
        "total_users": len(all_members),
        "users": all_members,
    }


async def list_organizations() -> dict:
    """
    List all organizations (tenants) in QuickPulse.
    Returns organization name, slug, plan, member count, and subscription status.
    Requires admin privileges.
    """
    data = await api_get("/api/v1/organizations")
    orgs = data if isinstance(data, list) else []
    return {
        "total": len(orgs),
        "organizations": [
            {
                "id": o.get("id"),
                "name": o.get("name"),
                "slug": o.get("slug"),
                "plan": o.get("plan"),
                "member_count": len(o.get("members", [])),
                "created_at": o.get("created_at"),
            }
            for o in orgs
        ],
    }


async def get_billing_overview() -> dict:
    """
    Get billing and subscription overview for the current organization.
    Returns current plan (free/pro/enterprise), subscription status (trial/active/cancelled),
    trial end date, resource limits (max hosts, users, containers), and usage.
    Requires admin privileges.
    """
    data = await api_get("/api/v1/billing/subscription")
    if not data:
        return {"error": "No billing data found. Organization may not have an active subscription."}

    plan = data.get("plan") or {}
    sub = data.get("subscription") or {}

    return {
        "plan": {
            "name": plan.get("name"),
            "display_name": plan.get("display_name"),
            "max_hosts": plan.get("max_hosts"),
            "max_users": plan.get("max_users"),
            "max_containers": plan.get("max_containers"),
            "metrics_retention_days": plan.get("metrics_retention_days"),
            "price_monthly_cents": plan.get("price_monthly_cents"),
        },
        "subscription": {
            "status": sub.get("status"),
            "trial_ends_at": sub.get("trial_ends_at"),
            "current_period_start": sub.get("current_period_start"),
            "current_period_end": sub.get("current_period_end"),
            "stripe_customer_id": sub.get("stripe_customer_id"),
            "cancelled_at": sub.get("cancelled_at"),
        },
    }
