"""
QuickPulse MCP Server
=====================
Exposes QuickPulse admin capabilities as Model Context Protocol (MCP) tools.
AI assistants (Claude, Cursor, etc.) can call these tools to manage containers,
review alerts, inspect metrics, and control the platform.

Authentication: Set QP_MCP_API_KEY environment variable to the same API key
you create in QuickPulse settings. All tool calls validate this key against
the backend API.

Usage (development):
    fastmcp dev mcp/server.py

Usage (production):
    python -m mcp.server
"""

import os
from fastmcp import FastMCP

from mcp.tools import containers, stacks, metrics, alerts, events, system, admin, kubernetes

# ── Server Entrypoint ────────────────────────────────────────────────────────

mcp = FastMCP(
    name="QuickPulse Admin MCP",
    instructions="""
    You are an AI assistant with full administrative access to a QuickPulse
    Docker monitoring and management platform.

    Available capabilities:
    - List, start, stop, restart containers and compose stacks
    - Fetch real-time host metrics (CPU, memory, disk, network)
    - Review and acknowledge alerts; manage alert rules
    - Browse Docker container events
    - Get system dashboard overview
    - Admin: list users, organizations, billing status

    Always confirm destructive actions (stop/restart) before proceeding.
    Use get_dashboard_summary first to understand the current system state.
    """,
)

# ── Register all tool modules ─────────────────────────────────────────────────

# Containers — list, inspect, start/stop/restart, get logs
mcp.tool(containers.list_containers)
mcp.tool(containers.get_container)
mcp.tool(containers.start_container)
mcp.tool(containers.stop_container)
mcp.tool(containers.restart_container)
mcp.tool(containers.get_container_logs)

# Stacks — docker compose stack management
mcp.tool(stacks.list_stacks)
mcp.tool(stacks.get_stack)
mcp.tool(stacks.start_stack)
mcp.tool(stacks.stop_stack)
mcp.tool(stacks.restart_stack)

# Metrics — host metrics and history
mcp.tool(metrics.get_current_metrics)
mcp.tool(metrics.get_metrics_history)

# Alerts — view and acknowledge alerts, manage rules
mcp.tool(alerts.list_active_alerts)
mcp.tool(alerts.acknowledge_alert)
mcp.tool(alerts.list_alert_rules)
mcp.tool(alerts.create_alert_rule)
mcp.tool(alerts.delete_alert_rule)

# Events — Docker container event log
mcp.tool(events.list_events)

# System — dashboard summary, host info
mcp.tool(system.get_dashboard_summary)
mcp.tool(system.get_host_info)

# Admin — users, organizations, billing (admin-only operations)
mcp.tool(admin.list_users)
mcp.tool(admin.list_organizations)
mcp.tool(admin.get_billing_overview)

# Kubernetes — cluster monitoring, pods, deployments, services, logs
mcp.tool(kubernetes.get_k8s_summary)
mcp.tool(kubernetes.list_k8s_nodes)
mcp.tool(kubernetes.list_k8s_pods)
mcp.tool(kubernetes.list_k8s_deployments)
mcp.tool(kubernetes.list_k8s_services)
mcp.tool(kubernetes.list_k8s_namespaces)
mcp.tool(kubernetes.list_k8s_events)
mcp.tool(kubernetes.get_k8s_pod_logs)


if __name__ == "__main__":
    mcp.run()
