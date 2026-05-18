import asyncio
import json
from collections import defaultdict
from typing import Any

import redis.asyncio as aioredis
from fastapi import WebSocket

from app.core.constants import WS_HEARTBEAT_INTERVAL
from app.core.logging import get_logger

logger = get_logger("websocket.manager")


class WebSocketManager:
    def __init__(self):
        self._connections: dict[str, set[WebSocket]] = defaultdict(set)
        self._redis: aioredis.Redis | None = None

    def set_redis(self, redis: aioredis.Redis | None) -> None:
        self._redis = redis

    async def connect(self, channel: str, websocket: WebSocket) -> None:
        try:
            await websocket.accept()
            self._connections[channel].add(websocket)
            logger.info("ws_connected", channel=channel, total=len(self._connections[channel]))
        except Exception as e:
            logger.error("ws_accept_failed", channel=channel, error=str(e), exc_info=True)

    def disconnect(self, channel: str, websocket: WebSocket) -> None:
        self._connections[channel].discard(websocket)
        if not self._connections[channel]:
            self._connections.pop(channel, None)
        logger.info("ws_disconnected", channel=channel)

    async def broadcast(self, channel: str, data: dict[str, Any]) -> None:
        if self._redis:
            try:
                await self._redis.publish(f"qp:{channel}", json.dumps(data, default=str))
            except Exception as e:
                logger.warning("ws_redis_publish_failed", channel=channel, error=str(e))
        await self._local_broadcast(channel, data)

    async def _local_broadcast(self, channel: str, data: dict[str, Any]) -> None:
        conns = list(self._connections.get(channel, set()))
        dead = set()
        for ws in conns:
            try:
                await ws.send_json(data)
            except Exception as e:
                logger.warning("ws_send_failed", channel=channel, error=str(e))
                dead.add(ws)
        for ws in dead:
            self._connections[channel].discard(ws)

    async def subscribe_to_redis_channels(self) -> None:
        if not self._redis:
            return
        try:
            pubsub = self._redis.pubsub()
            await pubsub.psubscribe("qp:*")
            async for message in pubsub.listen():
                if message["type"] == "pmessage":
                    channel = message["channel"]
                    if isinstance(channel, bytes):
                        channel = channel.decode()
                    local_channel = channel.replace("qp:", "", 1)
                    try:
                        data = json.loads(message["data"])
                        await self._local_broadcast(local_channel, data)
                    except Exception as e:
                        logger.warning("ws_redis_msg_malformed", channel=local_channel, error=str(e))
        except Exception as e:
            logger.error("ws_redis_subscribe_failed", error=str(e), exc_info=True)

    async def run_heartbeat(self) -> None:
        """Send periodic pings to keep WebSocket connections alive through NAT/LBs."""
        while True:
            try:
                await asyncio.sleep(WS_HEARTBEAT_INTERVAL)
                ping_msg = {"type": "ping"}
                for channel in list(self._connections.keys()):
                    await self._local_broadcast(channel, ping_msg)
                logger.debug("ws_heartbeat_sent", channels=list(self._connections.keys()))
            except asyncio.CancelledError:
                logger.info("ws_heartbeat_stopped")
                return
            except Exception as e:
                logger.warning("ws_heartbeat_error", error=str(e))

    def get_connection_count(self, channel: str) -> int:
        return len(self._connections.get(channel, set()))


ws_manager = WebSocketManager()
