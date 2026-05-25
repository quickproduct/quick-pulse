"""MCP tools for system-level overview and host information."""

from mcp.client import api_get


async def get_dashboard_summary() -> dict:
    """
    Get a high-level dashboard summary of the entire QuickPulse system.
    Returns:
    - Current host metrics (CPU, memory, disk)
    - Container count summary (total, running, stopped)
    - Stack health summary (running, partial, stopped stacks)
    - Active alert count
    - Recent events (last 5)
    This is the best starting point to understand overall system health.
    """
    data = await api_get("/api/v1/dashboard")
    if not data:
        return {"error": "Dashboard data unavailable"}

    metrics = data.get("metrics") or {}
    containers = data.get("containers") or {}
    stacks = data.get("stack_summary") or {}
    events = data.get("recent_events") or []

    critical_alerts = 0
    warning_alerts = 0
    try:
        alerts_data = await api_get("/api/v1/alerts")
        alerts_list = alerts_data if isinstance(alerts_data, list) else []
        active_list = [a for a in alerts_list if not a.get("acknowledged", False)]
        critical_alerts = sum(1 for a in active_list if a.get("severity") == "critical")
        warning_alerts = sum(1 for a in active_list if a.get("severity") == "warning")
    except Exception:
        pass

    cpu = metrics.get("cpu_percent", 0)
    mem = metrics.get("memory_percent", 0)

    if critical_alerts > 0 or cpu >= 90 or mem >= 90:
        health = "CRITICAL"
    elif warning_alerts > 0 or cpu >= 75 or mem >= 75:
        health = "WARNING"
    else:
        health = "HEALTHY"

    return {
        "overall_health": health,
        "metrics": {
            "cpu_percent": metrics.get("cpu_percent"),
            "memory_percent": metrics.get("memory_percent"),
            "disk_percent": metrics.get("disk_percent"),
            "load_1m": metrics.get("load_1m"),
        },
        "containers": {
            "total": containers.get("total", 0),
            "running": containers.get("running", 0),
            "stopped": containers.get("stopped", 0),
        },
        "stacks": {
            "total": stacks.get("total", 0),
            "running": stacks.get("running", 0),
            "partial": stacks.get("partial", 0),
            "stopped": stacks.get("stopped", 0),
        },
        "alerts": {
            "critical": critical_alerts,
            "warning": warning_alerts,
        },
        "recent_events": [
            {
                "container": e.get("container_name"),
                "event": e.get("event_type"),
                "time": e.get("timestamp"),
            }
            for e in events[:5]
        ],
    }


async def get_host_info() -> dict:
    """
    Get detailed host machine information.
    Returns hostname, IP address, OS, CPU count, total memory, total disk, and uptime.
    Useful for understanding the underlying hardware running your containers.
    """
    data = await api_get("/api/v1/dashboard")
    host = (data or {}).get("host") or {}
    metrics = (data or {}).get("metrics") or {}
    return {
        "hostname": host.get("hostname"),
        "ip_address": host.get("ip_address"),
        "os": host.get("os_info"),
        "cpu_count": host.get("cpu_count"),
        "total_memory_bytes": host.get("total_memory"),
        "total_disk_bytes": host.get("total_disk"),
        "uptime_seconds": metrics.get("uptime_seconds"),
    }
