"""MCP tools for Kubernetes cluster management and monitoring."""

from typing import Annotated
from mcp.client import api_get


async def get_k8s_summary() -> dict:
    """
    Get a high-level summary of the active Kubernetes cluster.
    Returns counts of nodes, namespaces, pods, deployments, and services, 
    along with basic health details.
    """
    return await api_get("/api/v1/kubernetes/summary")


async def list_k8s_nodes() -> list:
    """
    List all active nodes in the Kubernetes cluster.
    Returns node name, status, roles, version, internal/external IP, 
    and resource allocation.
    """
    return await api_get("/api/v1/kubernetes/nodes")


async def list_k8s_pods(
    namespace: Annotated[str | None, "Filter pods by namespace. If None, retrieves pods from all namespaces."] = None
) -> list:
    """
    List pods in the Kubernetes cluster.
    Can be filtered by namespace. Returns pod name, namespace, status, restarts, node, and age.
    """
    params = {}
    if namespace:
        params["namespace"] = namespace
    return await api_get("/api/v1/kubernetes/pods", params=params)


async def list_k8s_deployments(
    namespace: Annotated[str | None, "Filter deployments by namespace. If None, retrieves from all namespaces."] = None
) -> list:
    """
    List Kubernetes deployments.
    Can be filtered by namespace. Returns deployment name, namespace, replicas (desired/ready/available), and status.
    """
    params = {}
    if namespace:
        params["namespace"] = namespace
    return await api_get("/api/v1/kubernetes/deployments", params=params)


async def list_k8s_services(
    namespace: Annotated[str | None, "Filter services by namespace. If None, retrieves from all namespaces."] = None
) -> list:
    """
    List Kubernetes services.
    Can be filtered by namespace. Returns service name, namespace, type, cluster IP, external IPs, and ports.
    """
    params = {}
    if namespace:
        params["namespace"] = namespace
    return await api_get("/api/v1/kubernetes/services", params=params)


async def list_k8s_namespaces() -> list:
    """
    List all namespaces in the active Kubernetes cluster.
    """
    return await api_get("/api/v1/kubernetes/namespaces")


async def list_k8s_events(
    namespace: Annotated[str | None, "Filter events by namespace. If None, retrieves from all namespaces."] = None
) -> list:
    """
    Retrieve the event feed from the Kubernetes cluster.
    Can be filtered by namespace.
    """
    params = {}
    if namespace:
        params["namespace"] = namespace
    return await api_get("/api/v1/kubernetes/events", params=params)


async def get_k8s_pod_logs(
    namespace: Annotated[str, "The Kubernetes namespace of the pod"],
    pod_name: Annotated[str, "The name of the pod"],
    tail: Annotated[int, "Number of log lines to retrieve (1-500, default 100)"] = 100,
) -> dict:
    """
    Retrieve initial log output from the first container of a Kubernetes pod.
    """
    tail = max(1, min(tail, 500))
    data = await api_get(f"/api/v1/kubernetes/pods/{namespace}/{pod_name}/logs", params={"tail": tail})
    logs = data.get("logs", []) if isinstance(data, dict) else []
    return {
        "namespace": namespace,
        "pod_name": pod_name,
        "lines_returned": len(logs),
        "logs": logs,
    }
