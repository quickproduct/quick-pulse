"""MCP tools for alert management."""

from typing import Annotated, Literal
from mcp.client import api_get, api_post, api_delete


async def list_active_alerts() -> dict:
    """
    List all currently active (unacknowledged) alerts.
    Alerts are triggered when metrics exceed configured thresholds.
    Returns severity (info/warning/critical), message, rule ID, and timestamp.
    """
    data = await api_get("/api/v1/alerts")
    alerts = data if isinstance(data, list) else []
    active = [a for a in alerts if not a.get("acknowledged", False)]
    return {
        "total_active": len(active),
        "critical": sum(1 for a in active if a.get("severity") == "critical"),
        "warning": sum(1 for a in active if a.get("severity") == "warning"),
        "info": sum(1 for a in active if a.get("severity") == "info"),
        "alerts": active,
    }


async def acknowledge_alert(
    alert_id: Annotated[str, "The UUID of the alert to acknowledge"]
) -> dict:
    """
    Acknowledge an alert to mark it as reviewed and suppress further notifications.
    Requires admin privileges. The alert will no longer appear in the active alerts list.
    """
    return await api_post(f"/api/v1/alerts/{alert_id}/acknowledge")


async def list_alert_rules() -> dict:
    """
    List all configured alert rules.
    Rules define thresholds for metrics (CPU, memory, disk, etc.) that trigger alerts.
    Returns metric type, threshold value, operator (gt/gte/lt/lte/eq), and enabled status.
    """
    data = await api_get("/api/v1/alert-rules")
    rules = data if isinstance(data, list) else []
    return {
        "total": len(rules),
        "enabled": sum(1 for r in rules if r.get("enabled")),
        "rules": rules,
    }


async def create_alert_rule(
    metric_type: Annotated[
        Literal["cpu", "memory", "disk", "network", "load"],
        "The metric to monitor"
    ],
    threshold: Annotated[float, "The numeric threshold value that triggers the alert"],
    operator: Annotated[
        Literal["gt", "gte", "lt", "lte", "eq"],
        "Comparison operator: gt (greater than), gte (>=), lt (<), lte (<=), eq (==)"
    ] = "gte",
    duration_seconds: Annotated[
        int,
        "How many seconds the condition must persist before firing the alert (default 60)"
    ] = 60,
) -> dict:
    """
    Create a new alert rule to monitor a specific metric threshold.
    Example: Create a CPU alert that fires when CPU >= 85% for 60 seconds.
    Requires admin privileges.
    """
    return await api_post("/api/v1/alert-rules", json={
        "metric_type": metric_type,
        "threshold": threshold,
        "operator": operator,
        "duration_seconds": duration_seconds,
    })


async def delete_alert_rule(
    rule_id: Annotated[str, "The UUID of the alert rule to delete"]
) -> dict:
    """
    Delete an alert rule permanently. This will stop monitoring for the configured threshold.
    Requires admin privileges. Any existing alerts from this rule will remain but won't update.
    """
    await api_delete(f"/api/v1/alert-rules/{rule_id}")
    return {"success": True, "message": f"Alert rule {rule_id} deleted"}
