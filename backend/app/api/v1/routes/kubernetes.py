"""Kubernetes REST API routes."""

from fastapi import APIRouter, Depends, Query

from app.core.logging import get_logger
from app.domain.entities.user import User
from app.utils.deps import get_current_user
from app.infrastructure.kubernetes import client as k8s

router = APIRouter()
logger = get_logger("api.kubernetes")


@router.get("/overview")
async def cluster_overview(current_user: User = Depends(get_current_user)):
    """High-level cluster health: nodes, pods, namespace counts."""
    logger.info("k8s_overview", user_id=str(current_user.id))
    return await k8s.get_cluster_overview()


@router.get("/nodes")
async def list_nodes(current_user: User = Depends(get_current_user)):
    """List all cluster nodes with status, role, and resource capacity."""
    return await k8s.get_nodes()


@router.get("/pods")
async def list_pods(
    namespace: str | None = Query(None, description="Filter by namespace"),
    current_user: User = Depends(get_current_user),
):
    """List pods, optionally filtered by namespace."""
    return await k8s.get_pods(namespace=namespace)


@router.get("/deployments")
async def list_deployments(
    namespace: str | None = Query(None, description="Filter by namespace"),
    current_user: User = Depends(get_current_user),
):
    """List deployments with replica counts and rollout status."""
    return await k8s.get_deployments(namespace=namespace)


@router.get("/services")
async def list_services(
    namespace: str | None = Query(None, description="Filter by namespace"),
    current_user: User = Depends(get_current_user),
):
    """List services with type, ports, and selectors."""
    return await k8s.get_services(namespace=namespace)


@router.get("/namespaces")
async def list_namespaces(current_user: User = Depends(get_current_user)):
    """List all namespaces."""
    return await k8s.get_namespaces()


@router.get("/events")
async def list_events(
    namespace: str | None = Query(None, description="Filter by namespace"),
    current_user: User = Depends(get_current_user),
):
    """List cluster events (warnings + normal), newest first."""
    return await k8s.get_events(namespace=namespace)
