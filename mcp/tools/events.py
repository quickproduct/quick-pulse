"""MCP tools for Docker container event log."""

from typing import Annotated
from mcp.client import api_get


async def list_events(
    limit: Annotated[int, "Maximum number of events to return (1-100, default 50)"] = 50,
    container_name: Annotated[str | None, "Filter events by container name (optional)"] = None,
) -> dict:
    """
    List recent Docker container lifecycle events.
    Events include container_start, container_stop, container_restart, container_die,
    container_create, container_destroy, and container_health status changes.
    Returns events in reverse chronological order (newest first).
    """
    limit = max(1, min(limit, 100))
    params: dict = {"limit": limit}
    if container_name:
        params["container_name"] = container_name

    data = await api_get("/api/v1/events", params=params)
    events = data if isinstance(data, list) else (data.get("events", []) if isinstance(data, dict) else [])

    # Group by event type for a quick summary
    type_counts: dict[str, int] = {}
    for e in events:
        et = e.get("event_type", "unknown")
        type_counts[et] = type_counts.get(et, 0) + 1

    return {
        "total": len(events),
        "event_type_summary": type_counts,
        "events": [
            {
                "id": e.get("id"),
                "container_name": e.get("container_name"),
                "container_id": e.get("container_docker_id"),
                "event_type": e.get("event_type"),
                "timestamp": e.get("timestamp"),
            }
            for e in events
        ],
    }
