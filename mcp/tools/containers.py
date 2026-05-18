"""MCP tools for Docker container management."""

from typing import Annotated
from mcp.client import api_get, api_post


async def list_containers(
    show_all: Annotated[bool, "If True, include stopped containers. Default False (only running)."] = False
) -> dict:
    """
    List all Docker containers on the host.
    Returns container name, image, status, ports, and docker ID.
    Use show_all=True to include stopped/exited containers.
    """
    data = await api_get("/api/v1/containers", params={"all": show_all})
    containers = data if isinstance(data, list) else []
    summary = [
        {
            "id": c.get("docker_id"),
            "name": c.get("name"),
            "image": c.get("image"),
            "status": c.get("status"),
            "status_text": c.get("status_text"),
            "ports": c.get("ports", []),
        }
        for c in containers
    ]
    return {
        "total": len(summary),
        "running": sum(1 for c in summary if c["status"] == "running"),
        "containers": summary,
    }


async def get_container(
    container_id: Annotated[str, "The Docker container ID (short 12-char or full ID)"]
) -> dict:
    """
    Inspect a specific Docker container in detail.
    Returns full container metadata including network settings, mounts, env vars, and resource limits.
    """
    return await api_get(f"/api/v1/containers/{container_id}")


async def start_container(
    container_id: Annotated[str, "The Docker container ID to start"]
) -> dict:
    """
    Start a stopped Docker container.
    Requires admin privileges. Returns success status and message.
    CAUTION: Confirm the container name/ID before starting.
    """
    return await api_post(f"/api/v1/containers/{container_id}/start")


async def stop_container(
    container_id: Annotated[str, "The Docker container ID to stop"]
) -> dict:
    """
    Stop a running Docker container gracefully (SIGTERM, then SIGKILL after timeout).
    Requires admin privileges.
    CAUTION: This will interrupt any active workloads in the container.
    """
    return await api_post(f"/api/v1/containers/{container_id}/stop")


async def restart_container(
    container_id: Annotated[str, "The Docker container ID to restart"]
) -> dict:
    """
    Restart a Docker container (stop + start).
    Requires admin privileges. Useful for applying config changes without destroying the container.
    CAUTION: Causes a brief service interruption.
    """
    return await api_post(f"/api/v1/containers/{container_id}/restart")


async def get_container_logs(
    container_id: Annotated[str, "The Docker container ID"],
    tail: Annotated[int, "Number of log lines to return (1-500, default 100)"] = 100,
) -> dict:
    """
    Fetch the most recent log output from a Docker container (stdout + stderr).
    Returns the last N lines. Use tail to control how many lines to fetch.
    """
    tail = max(1, min(tail, 500))
    data = await api_get(f"/api/v1/containers/{container_id}/logs", params={"tail": tail})
    logs = data.get("logs", []) if isinstance(data, dict) else []
    return {
        "container_id": container_id,
        "lines_returned": len(logs),
        "logs": logs,
    }
