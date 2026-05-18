"""
Kubernetes Client
=================
Tries to connect to a real Kubernetes cluster (in-cluster config first,
then kubeconfig). If neither is available, falls back to returning
realistic mock data so the dashboard always renders something useful.
"""

import os
import json
import random
from datetime import datetime, timezone, timedelta
from typing import Any

from app.core.logging import get_logger

logger = get_logger("kubernetes.client")

_k8s_available = False
_core_v1 = None
_apps_v1 = None


def _try_init_real_client() -> bool:
    """Attempt to initialise the official kubernetes Python client."""
    try:
        from kubernetes import client, config  # type: ignore
        try:
            config.load_incluster_config()
            logger.info("k8s_incluster_config_loaded")
        except Exception:
            kubeconfig = os.environ.get("KUBECONFIG", os.path.expanduser("~/.kube/config"))
            config.load_kube_config(config_file=kubeconfig)
            logger.info("k8s_kubeconfig_loaded", path=kubeconfig)

        global _core_v1, _apps_v1
        
        # Check for server override (essential for Docker-to-Host cluster connectivity)
        server_override = os.environ.get("K8S_SERVER_OVERRIDE")
        if server_override:
            # We must modify the configuration after loading it
            from kubernetes.client import Configuration
            conf = Configuration.get_default_copy()
            conf.host = server_override
            # Local k3d clusters often have certs issued for 127.0.0.1; 
            # bypass verify for host.docker.internal access.
            conf.verify_ssl = False
            conf.assert_hostname = False
            
            # Set as default for any client instantiated without explicit api_client
            Configuration.set_default(conf)
            
            api_client = client.ApiClient(conf)
            _core_v1 = client.CoreV1Api(api_client)
            _apps_v1 = client.AppsV1Api(api_client)
            logger.info("k8s_client_initialized_with_override", server=server_override)
        else:
            _core_v1 = client.CoreV1Api()
            _apps_v1 = client.AppsV1Api()
            
        return True
    except Exception as e:
        logger.warning("k8s_client_unavailable", error=str(e),
                       detail="Kubernetes dashboard will use mock data")
        return False


_k8s_available = _try_init_real_client()


# ── Helpers ────────────────────────────────────────────────────────────────────

def _ago(seconds: int) -> str:
    dt = datetime.now(timezone.utc) - timedelta(seconds=seconds)
    return dt.isoformat()


def _rand_cpu() -> str:
    return f"{random.randint(1, 450)}m"


def _rand_mem() -> str:
    return f"{random.randint(32, 512)}Mi"


# ── Mock data ──────────────────────────────────────────────────────────────────

_NAMESPACES = ["default", "kube-system", "monitoring", "production", "staging"]
_NODES = [
    {"name": "node-01", "role": "control-plane", "status": "Ready", "version": "v1.29.3",
     "cpu": "4", "memory": "8Gi", "os": "Ubuntu 22.04", "arch": "amd64",
     "age_seconds": 3_600_000, "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "node-02", "role": "worker", "status": "Ready", "version": "v1.29.3",
     "cpu": "8", "memory": "16Gi", "os": "Ubuntu 22.04", "arch": "amd64",
     "age_seconds": 3_598_000, "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "node-03", "role": "worker", "status": "Ready", "version": "v1.29.3",
     "cpu": "8", "memory": "16Gi", "os": "Ubuntu 22.04", "arch": "amd64",
     "age_seconds": 3_550_000, "conditions": [{"type": "Ready", "status": "True"}]},
]

_PODS = [
    {"name": "quickpulse-backend-7d9f8b-kxp2l", "namespace": "production", "status": "Running",
     "ready": "1/1", "restarts": 0, "age_seconds": 86400, "node": "node-02",
     "cpu": "45m", "memory": "128Mi", "image": "quickpulse/backend:latest",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "quickpulse-frontend-5c7d9f-mn4rs", "namespace": "production", "status": "Running",
     "ready": "1/1", "restarts": 0, "age_seconds": 86400, "node": "node-03",
     "cpu": "12m", "memory": "64Mi", "image": "quickpulse/frontend:latest",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "quickpulse-db-0", "namespace": "production", "status": "Running",
     "ready": "1/1", "restarts": 0, "age_seconds": 172800, "node": "node-02",
     "cpu": "180m", "memory": "512Mi", "image": "timescale/timescaledb:latest-pg16",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "quickpulse-redis-0", "namespace": "production", "status": "Running",
     "ready": "1/1", "restarts": 1, "age_seconds": 172800, "node": "node-03",
     "cpu": "8m", "memory": "32Mi", "image": "redis:7-alpine",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "prometheus-0", "namespace": "monitoring", "status": "Running",
     "ready": "1/1", "restarts": 0, "age_seconds": 604800, "node": "node-02",
     "cpu": "220m", "memory": "384Mi", "image": "prom/prometheus:v2.52.0",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "grafana-6f8b9d-wr5kp", "namespace": "monitoring", "status": "Running",
     "ready": "1/1", "restarts": 0, "age_seconds": 604800, "node": "node-03",
     "cpu": "35m", "memory": "96Mi", "image": "grafana/grafana:10.4.2",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "coredns-7db6d8ff4-nt9zp", "namespace": "kube-system", "status": "Running",
     "ready": "2/2", "restarts": 0, "age_seconds": 3_600_000, "node": "node-01",
     "cpu": "6m", "memory": "18Mi", "image": "registry.k8s.io/coredns/coredns:v1.11.3",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "kube-apiserver-node-01", "namespace": "kube-system", "status": "Running",
     "ready": "1/1", "restarts": 2, "age_seconds": 3_600_000, "node": "node-01",
     "cpu": "95m", "memory": "256Mi", "image": "registry.k8s.io/kube-apiserver:v1.29.3",
     "conditions": [{"type": "Ready", "status": "True"}]},
    {"name": "staging-api-deploy-78c9d-qp2kr", "namespace": "staging", "status": "Pending",
     "ready": "0/1", "restarts": 0, "age_seconds": 120, "node": "",
     "cpu": "0m", "memory": "0Mi", "image": "quickpulse/backend:edge",
     "conditions": [{"type": "PodScheduled", "status": "False", "reason": "Unschedulable"}]},
    {"name": "batch-job-xkcd9-wrkr", "namespace": "default", "status": "Failed",
     "ready": "0/1", "restarts": 5, "age_seconds": 3600, "node": "node-02",
     "cpu": "0m", "memory": "0Mi", "image": "quickpulse/batch:1.2.0",
     "conditions": [{"type": "Ready", "status": "False"}]},
]

_DEPLOYMENTS = [
    {"name": "quickpulse-backend", "namespace": "production",
     "desired": 2, "ready": 2, "available": 2, "updated": 2,
     "age_seconds": 86400, "image": "quickpulse/backend:latest", "strategy": "RollingUpdate"},
    {"name": "quickpulse-frontend", "namespace": "production",
     "desired": 2, "ready": 2, "available": 2, "updated": 2,
     "age_seconds": 86400, "image": "quickpulse/frontend:latest", "strategy": "RollingUpdate"},
    {"name": "grafana", "namespace": "monitoring",
     "desired": 1, "ready": 1, "available": 1, "updated": 1,
     "age_seconds": 604800, "image": "grafana/grafana:10.4.2", "strategy": "Recreate"},
    {"name": "coredns", "namespace": "kube-system",
     "desired": 2, "ready": 2, "available": 2, "updated": 2,
     "age_seconds": 3_600_000, "image": "registry.k8s.io/coredns/coredns:v1.11.3", "strategy": "RollingUpdate"},
    {"name": "staging-api-deploy", "namespace": "staging",
     "desired": 3, "ready": 1, "available": 1, "updated": 3,
     "age_seconds": 7200, "image": "quickpulse/backend:edge", "strategy": "RollingUpdate"},
]

_SERVICES = [
    {"name": "quickpulse-backend", "namespace": "production", "type": "ClusterIP",
     "cluster_ip": "10.96.45.12", "external_ip": None,
     "ports": [{"port": 8000, "target_port": 8000, "protocol": "TCP"}],
     "selector": {"app": "quickpulse-backend"}, "age_seconds": 86400},
    {"name": "quickpulse-frontend", "namespace": "production", "type": "LoadBalancer",
     "cluster_ip": "10.96.45.13", "external_ip": "203.0.113.42",
     "ports": [{"port": 80, "target_port": 80, "protocol": "TCP"}],
     "selector": {"app": "quickpulse-frontend"}, "age_seconds": 86400},
    {"name": "quickpulse-db", "namespace": "production", "type": "ClusterIP",
     "cluster_ip": "10.96.45.14", "external_ip": None,
     "ports": [{"port": 5432, "target_port": 5432, "protocol": "TCP"}],
     "selector": {"app": "quickpulse-db"}, "age_seconds": 172800},
    {"name": "quickpulse-redis", "namespace": "production", "type": "ClusterIP",
     "cluster_ip": "10.96.45.15", "external_ip": None,
     "ports": [{"port": 6379, "target_port": 6379, "protocol": "TCP"}],
     "selector": {"app": "quickpulse-redis"}, "age_seconds": 172800},
    {"name": "prometheus", "namespace": "monitoring", "type": "NodePort",
     "cluster_ip": "10.96.78.10", "external_ip": None,
     "ports": [{"port": 9090, "target_port": 9090, "node_port": 30090, "protocol": "TCP"}],
     "selector": {"app": "prometheus"}, "age_seconds": 604800},
    {"name": "grafana", "namespace": "monitoring", "type": "NodePort",
     "cluster_ip": "10.96.78.11", "external_ip": None,
     "ports": [{"port": 3000, "target_port": 3000, "node_port": 30030, "protocol": "TCP"}],
     "selector": {"app": "grafana"}, "age_seconds": 604800},
    {"name": "kubernetes", "namespace": "default", "type": "ClusterIP",
     "cluster_ip": "10.96.0.1", "external_ip": None,
     "ports": [{"port": 443, "target_port": 6443, "protocol": "TCP"}],
     "selector": {}, "age_seconds": 3_600_000},
]

_EVENTS = [
    {"name": "quickpulse-backend.17d1f2a9b3", "namespace": "production",
     "type": "Normal", "reason": "Pulled", "object": "Pod/quickpulse-backend-7d9f8b-kxp2l",
     "message": "Successfully pulled image \"quickpulse/backend:latest\" in 2.341s",
     "count": 1, "age_seconds": 86400},
    {"name": "quickpulse-backend.17d1f2b3c4", "namespace": "production",
     "type": "Normal", "reason": "Started", "object": "Pod/quickpulse-backend-7d9f8b-kxp2l",
     "message": "Started container quickpulse-backend",
     "count": 1, "age_seconds": 86395},
    {"name": "staging-api.17d1f2d8e9", "namespace": "staging",
     "type": "Warning", "reason": "FailedScheduling", "object": "Pod/staging-api-deploy-78c9d-qp2kr",
     "message": "0/3 nodes are available: 3 Insufficient cpu. preemption: 0/3 nodes are available: 3 No preemption victims found for incoming pod.",
     "count": 12, "age_seconds": 120},
    {"name": "batch-job.17d1f1a2b3", "namespace": "default",
     "type": "Warning", "reason": "BackOff", "object": "Pod/batch-job-xkcd9-wrkr",
     "message": "Back-off restarting failed container batch-worker in pod batch-job-xkcd9-wrkr_default",
     "count": 5, "age_seconds": 3600},
    {"name": "quickpulse-redis.17d1f0c5d6", "namespace": "production",
     "type": "Normal", "reason": "Killing", "object": "Pod/quickpulse-redis-0",
     "message": "Stopping container redis",
     "count": 1, "age_seconds": 172800},
]


# ── Public API ─────────────────────────────────────────────────────────────────

async def get_cluster_overview() -> dict:
    if _k8s_available and _core_v1:
        try:
            nodes = _core_v1.list_node()
            pods = _core_v1.list_pod_for_all_namespaces()
            logger.debug("k8s_overview_success", source="live", pods=len(pods.items))
            return {
                "nodes": len(nodes.items),
                "nodes_ready": sum(1 for n in nodes.items
                                   if any(c.type == "Ready" and c.status == "True"
                                          for c in n.status.conditions)),
                "pods_total": len(pods.items),
                "pods_running": sum(1 for p in pods.items if p.status.phase == "Running"),
                "pods_pending": sum(1 for p in pods.items if p.status.phase == "Pending"),
                "pods_failed": sum(1 for p in pods.items if p.status.phase == "Failed"),
                "namespaces": len(set(p.metadata.namespace for p in pods.items)),
                "source": "live",
            }
        except Exception as e:
            logger.warning("k8s_overview_failed", error=str(e))

    # Mock
    pods = _PODS
    return {
        "nodes": len(_NODES),
        "nodes_ready": sum(1 for n in _NODES if n["status"] == "Ready"),
        "pods_total": len(pods),
        "pods_running": sum(1 for p in pods if p["status"] == "Running"),
        "pods_pending": sum(1 for p in pods if p["status"] == "Pending"),
        "pods_failed": sum(1 for p in pods if p["status"] == "Failed"),
        "namespaces": len(_NAMESPACES),
        "source": "mock",
    }


async def get_nodes() -> list[dict]:
    if _k8s_available and _core_v1:
        try:
            raw = _core_v1.list_node()
            result = []
            for n in raw.items:
                roles = [l.replace("node-role.kubernetes.io/", "") for l in n.metadata.labels
                         if "node-role.kubernetes.io/" in l] or ["worker"]
                ready = next(
                    (c.status for c in n.status.conditions if c.type == "Ready"), "Unknown"
                )
                result.append({
                    "name": n.metadata.name,
                    "role": ",".join(roles),
                    "status": "Ready" if ready == "True" else "NotReady",
                    "version": n.status.node_info.kubelet_version,
                    "cpu": n.status.capacity.get("cpu"),
                    "memory": n.status.capacity.get("memory"),
                    "os": n.status.node_info.os_image,
                    "arch": n.status.node_info.architecture,
                    "age_seconds": int((datetime.now(timezone.utc) - n.metadata.creation_timestamp.replace(tzinfo=timezone.utc)).total_seconds()),
                })
            return result
        except Exception as e:
            logger.warning("k8s_nodes_failed", error=str(e))

    return _NODES


async def get_pods(namespace: str | None = None) -> list[dict]:
    if _k8s_available and _core_v1:
        try:
            if namespace:
                raw = _core_v1.list_namespaced_pod(namespace)
            else:
                raw = _core_v1.list_pod_for_all_namespaces()
            result = []
            for p in raw.items:
                restarts = sum(cs.restart_count for cs in (p.status.container_statuses or []))
                result.append({
                    "name": p.metadata.name,
                    "namespace": p.metadata.namespace,
                    "status": p.status.phase or "Unknown",
                    "ready": f"{sum(1 for cs in (p.status.container_statuses or []) if cs.ready)}/{len(p.spec.containers)}",
                    "restarts": restarts,
                    "node": p.spec.node_name or "",
                    "age_seconds": int((datetime.now(timezone.utc) - p.metadata.creation_timestamp.replace(tzinfo=timezone.utc)).total_seconds()),
                    "image": p.spec.containers[0].image if p.spec.containers else "",
                })
            return result
        except Exception as e:
            logger.warning("k8s_pods_failed", error=str(e))

    pods = _PODS
    if namespace:
        pods = [p for p in pods if p["namespace"] == namespace]
    return pods


async def get_deployments(namespace: str | None = None) -> list[dict]:
    if _k8s_available and _apps_v1:
        try:
            if namespace:
                raw = _apps_v1.list_namespaced_deployment(namespace)
            else:
                raw = _apps_v1.list_deployment_for_all_namespaces()
            result = []
            for d in raw.items:
                spec = d.spec or {}
                status = d.status or {}
                containers = (spec.template.spec.containers or []) if hasattr(spec, "template") else []
                result.append({
                    "name": d.metadata.name,
                    "namespace": d.metadata.namespace,
                    "desired": spec.replicas or 0,
                    "ready": status.ready_replicas or 0,
                    "available": status.available_replicas or 0,
                    "updated": status.updated_replicas or 0,
                    "age_seconds": int((datetime.now(timezone.utc) - d.metadata.creation_timestamp.replace(tzinfo=timezone.utc)).total_seconds()),
                    "image": containers[0].image if containers else "",
                    "strategy": d.spec.strategy.type if d.spec.strategy else "RollingUpdate",
                })
            return result
        except Exception as e:
            logger.warning("k8s_deployments_failed", error=str(e))

    deployments = _DEPLOYMENTS
    if namespace:
        deployments = [d for d in deployments if d["namespace"] == namespace]
    return deployments


async def get_services(namespace: str | None = None) -> list[dict]:
    if _k8s_available and _core_v1:
        try:
            if namespace:
                raw = _core_v1.list_namespaced_service(namespace)
            else:
                raw = _core_v1.list_service_for_all_namespaces()
            result = []
            for s in raw.items:
                spec = s.spec or {}
                result.append({
                    "name": s.metadata.name,
                    "namespace": s.metadata.namespace,
                    "type": spec.type or "ClusterIP",
                    "cluster_ip": spec.cluster_ip,
                    "external_ip": (spec.external_i_ps or [None])[0] if hasattr(spec, "external_i_ps") else None,
                    "ports": [
                        {"port": p.port, "target_port": p.target_port, "protocol": p.protocol}
                        for p in (spec.ports or [])
                    ],
                    "selector": spec.selector or {},
                    "age_seconds": int((datetime.now(timezone.utc) - s.metadata.creation_timestamp.replace(tzinfo=timezone.utc)).total_seconds()),
                })
            return result
        except Exception as e:
            logger.warning("k8s_services_failed", error=str(e))

    services = _SERVICES
    if namespace:
        services = [s for s in services if s["namespace"] == namespace]
    return services


async def get_namespaces() -> list[str]:
    if _k8s_available and _core_v1:
        try:
            raw = _core_v1.list_namespace()
            return [n.metadata.name for n in raw.items]
        except Exception as e:
            logger.warning("k8s_namespaces_failed", error=str(e))
    return _NAMESPACES


async def get_events(namespace: str | None = None) -> list[dict]:
    if _k8s_available and _core_v1:
        try:
            if namespace:
                raw = _core_v1.list_namespaced_event(namespace)
            else:
                raw = _core_v1.list_event_for_all_namespaces()
            result = []
            for e in raw.items:
                result.append({
                    "name": e.metadata.name,
                    "namespace": e.metadata.namespace,
                    "type": e.type or "Normal",
                    "reason": e.reason or "",
                    "object": f"{e.involved_object.kind}/{e.involved_object.name}",
                    "message": e.message or "",
                    "count": e.count or 1,
                    "age_seconds": int((datetime.now(timezone.utc) - (e.last_timestamp or e.metadata.creation_timestamp).replace(tzinfo=timezone.utc)).total_seconds()),
                })
            return sorted(result, key=lambda x: x["age_seconds"])
        except Exception as e:
            logger.warning("k8s_events_failed", error=str(e))

    events = _EVENTS
    if namespace:
        events = [e for e in events if e["namespace"] == namespace]
    return events
