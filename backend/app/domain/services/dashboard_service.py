from typing import Any

from app.core.logging import get_logger
from app.domain.interfaces.repositories import ContainerEventRepository, ContainerRepository, HostRepository, MetricsRepository, StackRepository
from app.domain.services.compose_service import ComposeService

logger = get_logger("dashboard_service")


class DashboardService:
    def __init__(
        self,
        docker: Any,
        host_repo: HostRepository,
        metrics_repo: MetricsRepository,
        container_repo: ContainerRepository,
        event_repo: ContainerEventRepository,
        stack_repo: StackRepository,
    ):
        self._docker = docker
        self._host_repo = host_repo
        self._metrics_repo = metrics_repo
        self._container_repo = container_repo
        self._event_repo = event_repo
        self._stack_repo = stack_repo

    async def get_dashboard(self) -> dict:
        # Host & metrics
        host = None
        metrics = None
        try:
            host = await self._host_repo.get_current()
            if host:
                latest = await self._metrics_repo.get_latest(host.id)
                if latest:
                    metrics = {
                        "cpu_percent": latest.cpu_percent,
                        "memory_percent": latest.memory_percent,
                        "memory_used": latest.memory_used,
                        "memory_total": latest.memory_total,
                        "disk_percent": latest.disk_percent,
                        "net_bytes_sent": latest.net_bytes_sent,
                        "net_bytes_recv": latest.net_bytes_recv,
                        "load_1m": latest.load_1m,
                        "load_5m": latest.load_5m,
                        "load_15m": latest.load_15m,
                        "process_count": latest.process_count,
                        "uptime_seconds": latest.uptime_seconds,
                    }
        except Exception as e:
            logger.error("dashboard_host_metrics_failed", error=str(e), exc_info=True)

        # Container summary
        container_summary = {"total": 0, "running": 0, "stopped": 0}
        try:
            containers = await self._docker.containers.list(all=True)
            filtered = []
            for c in containers:
                raw = c._container or {}
                labels = raw.get("Labels") or {}
                project = labels.get("com.docker.compose.project", "")
                names = raw.get("Names") or []
                name = names[0].lstrip("/") if names else (c.id or "")[:12]
                
                if name.startswith("qp-") or project == "quickpulse":
                    continue
                filtered.append(c)
                
            running = sum(1 for c in filtered if (c._container or {}).get("State") == "running")
            container_summary = {
                "total": len(filtered),
                "running": running,
                "stopped": len(filtered) - running,
            }
        except Exception as e:
            logger.warning("dashboard_containers_failed", error=str(e))

        # Recent events
        recent_events = []
        try:
            recent_events_raw = await self._event_repo.get_recent(limit=10)
            recent_events = [
                {
                    "id": str(e.id),
                    "container_name": e.container_name,
                    "container_docker_id": e.container_docker_id,
                    "event_type": e.event_type.value if hasattr(e.event_type, "value") else str(e.event_type),
                    "timestamp": e.timestamp.isoformat() if e.timestamp else None,
                }
                for e in recent_events_raw
            ]
        except Exception as e:
            logger.warning("dashboard_events_failed", error=str(e))

        # Stack summary
        stack_summary = {"total": 0, "running": 0, "partial": 0, "stopped": 0}
        try:
            cs = ComposeService(self._docker, self._stack_repo, None)
            stacks = await cs.list_stacks()
            stack_summary["total"] = len(stacks)
            for s in stacks:
                status = s.get("status", "unknown")
                if status in stack_summary:
                    stack_summary[status] += 1
        except Exception as e:
            logger.warning("dashboard_stacks_failed", error=str(e))

        return {
            "host": {
                "hostname": host.hostname if host else "unknown",
                "ip_address": host.ip_address if host else "unknown",
                "os_info": host.os_info if host else None,
                "cpu_count": host.cpu_count if host else 0,
                "total_memory": host.total_memory if host else 0,
                "total_disk": host.total_disk if host else 0,
            } if host else None,
            "metrics": metrics,
            "containers": container_summary,
            "recent_events": recent_events,
            "stack_summary": stack_summary,
        }
