import time

import psutil

from app.core.logging import get_logger

logger = get_logger("metrics.collector")


def collect_host_metrics() -> dict:
    try:
        cpu_percent = psutil.cpu_percent(interval=0.5)
        memory = psutil.virtual_memory()
        disk = psutil.disk_usage("/")
        disk_io = psutil.disk_io_counters()
        net_io = psutil.net_io_counters()
        load_avg = psutil.getloadavg()
        process_count = len(psutil.pids())
        uptime_seconds = int(time.time() - psutil.boot_time())

        return {
            "cpu_percent": round(cpu_percent, 2),
            "memory_percent": round(memory.percent, 2),
            "memory_used": memory.used,
            "memory_total": memory.total,
            "disk_percent": round(disk.percent, 2),
            "disk_read_bytes": disk_io.read_bytes if disk_io else 0,
            "disk_write_bytes": disk_io.write_bytes if disk_io else 0,
            "net_bytes_sent": net_io.bytes_sent if net_io else 0,
            "net_bytes_recv": net_io.bytes_recv if net_io else 0,
            "load_1m": round(load_avg[0], 2),
            "load_5m": round(load_avg[1], 2),
            "load_15m": round(load_avg[2], 2),
            "process_count": process_count,
            "uptime_seconds": uptime_seconds,
        }
    except Exception as e:
        logger.error("failed_to_collect_metrics", error=str(e), exc_info=True)
        return {}
