"""MCP tools for Docker Compose stack management."""

from typing import Annotated
from mcp.client import api_get, api_post


async def list_stacks() -> dict:
    """
    List all Docker Compose stacks running on the host.
    A stack is a group of containers managed by a docker-compose.yml file.
    Returns each stack's name, status (running/stopped/partial), service count, and services.
    """
    data = await api_get("/api/v1/stacks")
    stacks = data if isinstance(data, list) else []
    return {
        "total": len(stacks),
        "running": sum(1 for s in stacks if s.get("status") == "running"),
        "stacks": [
            {
                "name": s.get("name"),
                "status": s.get("status"),
                "services_count": s.get("services_count", 0),
                "running_services": s.get("running", 0),
                "project_dir": s.get("project_dir"),
            }
            for s in stacks
        ],
    }


async def get_stack(
    name: Annotated[str, "The Docker Compose project/stack name"]
) -> dict:
    """
    Get detailed information about a specific Docker Compose stack.
    Returns all services within the stack, their individual container IDs and statuses.
    """
    return await api_get(f"/api/v1/stacks/{name}")


async def start_stack(
    name: Annotated[str, "The Docker Compose stack name to start"]
) -> dict:
    """
    Start all stopped services in a Docker Compose stack.
    Requires admin privileges. Only starts services that are not already running.
    CAUTION: Confirm the stack name before starting.
    """
    return await api_post(f"/api/v1/stacks/{name}/start")


async def stop_stack(
    name: Annotated[str, "The Docker Compose stack name to stop"]
) -> dict:
    """
    Stop all running services in a Docker Compose stack.
    Requires admin privileges. Stops all containers in the compose project.
    CAUTION: This will interrupt all services in the stack.
    """
    return await api_post(f"/api/v1/stacks/{name}/stop")


async def restart_stack(
    name: Annotated[str, "The Docker Compose stack name to restart"]
) -> dict:
    """
    Restart all services in a Docker Compose stack.
    Requires admin privileges. Causes a brief interruption to all services.
    Useful after config changes or to resolve stuck containers.
    """
    return await api_post(f"/api/v1/stacks/{name}/restart")
