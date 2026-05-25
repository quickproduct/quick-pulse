"""MCP tools for host metrics retrieval."""

from typing import Annotated, Literal
from mcp.client import api_get


async def get_current_metrics() -> dict:
    """
    Get the latest real-time host metrics snapshot.
    Returns CPU usage %, memory usage %, disk usage %, network I/O bytes,
    system load averages (1m/5m/15m), process count, and uptime.
    Metrics are collected every 10 seconds by the metrics worker.
    """
    data = await api_get("/api/v1/metrics/summary")
    if not data:
        return {"error": "No metrics available yet. The metrics worker may still be initializing."}

    cpu = data.get("cpu_percent", 0)
    mem = data.get("memory_percent", 0)
    disk = data.get("disk_percent", 0)

    # Provide an interpretation of resource pressure
    def pressure(val: float) -> str:
        if val >= 90:
            return "CRITICAL"
        if val >= 75:
            return "HIGH"
        if val >= 50:
            return "MODERATE"
        return "NORMAL"

    return {
        "timestamp": data.get("timestamp"),
        "cpu": {"percent": cpu, "pressure": pressure(cpu)},
        "memory": {
            "percent": mem,
            "used_bytes": data.get("memory_used"),
            "total_bytes": data.get("memory_total"),
            "pressure": pressure(mem),
        },
        "disk": {"percent": disk, "pressure": pressure(disk)},
        "network": {
            "bytes_sent": data.get("net_bytes_sent"),
            "bytes_recv": data.get("net_bytes_recv"),
        },
        "load": {
            "1m": data.get("load_1m"),
            "5m": data.get("load_5m"),
            "15m": data.get("load_15m"),
        },
        "processes": data.get("process_count"),
        "uptime_seconds": data.get("uptime_seconds"),
    }


async def get_metrics_history(
    range: Annotated[
        Literal["1h", "24h", "7d"],
        "Time range for metrics history: '1h' (last hour), '24h' (last day), '7d' (last week)"
    ] = "1h"
) -> dict:
    """
    Get historical host metrics aggregated over a time range.
    Returns time-series data for CPU, memory, disk, and network.
    Useful for identifying trends, spikes, and performance regressions.
    """
    data = await api_get("/api/v1/metrics/history", params={"range": range})
    if isinstance(data, list):
        return {
            "range": range,
            "data_points": len(data),
            "metrics": data,
        }
    return data or {"range": range, "data_points": 0, "metrics": []}
