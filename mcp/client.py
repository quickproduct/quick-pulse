"""Shared HTTP client for MCP tools — calls the QuickPulse REST API."""

import os
import httpx
from typing import Any

_BASE_URL = os.environ.get("QP_API_URL", "http://localhost:8000")
_API_KEY = os.environ.get("QP_MCP_API_KEY", "")
_ACCESS_TOKEN: str | None = None


def _headers() -> dict[str, str]:
    """Return auth headers. Uses Bearer token if available, else API key header."""
    if _ACCESS_TOKEN:
        return {"Authorization": f"Bearer {_ACCESS_TOKEN}", "Content-Type": "application/json"}
    if _API_KEY:
        return {"X-API-Key": _API_KEY, "Content-Type": "application/json"}
    return {"Content-Type": "application/json"}


async def _login() -> None:
    """Auto-login using QP_ADMIN_EMAIL / QP_ADMIN_PASSWORD env vars."""
    global _ACCESS_TOKEN
    email = os.environ.get("QP_ADMIN_EMAIL", "admin@quickpulse.local")
    password = os.environ.get("QP_ADMIN_PASSWORD", "changeme")
    async with httpx.AsyncClient(base_url=_BASE_URL, timeout=10) as client:
        resp = await client.post("/api/v1/auth/login", json={"email": email, "password": password})
        resp.raise_for_status()
        _ACCESS_TOKEN = resp.json()["access_token"]


async def api_get(path: str, params: dict | None = None) -> Any:
    """Authenticated GET request to the QuickPulse API."""
    global _ACCESS_TOKEN
    async with httpx.AsyncClient(base_url=_BASE_URL, timeout=30) as client:
        resp = await client.get(path, params=params, headers=_headers())
        if resp.status_code == 401:
            await _login()
            resp = await client.get(path, params=params, headers=_headers())
        resp.raise_for_status()
        if resp.status_code == 204:
            return None
        return resp.json()


async def api_post(path: str, json: dict | None = None) -> Any:
    """Authenticated POST request to the QuickPulse API."""
    global _ACCESS_TOKEN
    async with httpx.AsyncClient(base_url=_BASE_URL, timeout=30) as client:
        resp = await client.post(path, json=json, headers=_headers())
        if resp.status_code == 401:
            await _login()
            resp = await client.post(path, json=json, headers=_headers())
        resp.raise_for_status()
        if resp.status_code == 204:
            return None
        return resp.json()


async def api_delete(path: str) -> Any:
    """Authenticated DELETE request to the QuickPulse API."""
    global _ACCESS_TOKEN
    async with httpx.AsyncClient(base_url=_BASE_URL, timeout=30) as client:
        resp = await client.delete(path, headers=_headers())
        if resp.status_code == 401:
            await _login()
            resp = await client.delete(path, headers=_headers())
        resp.raise_for_status()
        if resp.status_code == 204:
            return None
        return resp.json()
