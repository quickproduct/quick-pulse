from fastapi import APIRouter, WebSocket, WebSocketDisconnect, Depends
from app.infrastructure.websocket.manager import ws_manager
from app.infrastructure.kubernetes import client as k8s
from app.core.logging import get_logger
import asyncio
import json

router = APIRouter()
logger = get_logger("ws.kubernetes")

@router.websocket("")
async def kubernetes_ws(websocket: WebSocket):
    """
    WebSocket endpoint for real-time Kubernetes cluster updates.
    Streams cluster overview and event counts every 5 seconds.
    """
    await ws_manager.connect(websocket, "kubernetes")
    logger.info("k8s_ws_connected", client=websocket.client)
    
    try:
        while True:
            # Fetch latest overview
            overview = await k8s.get_cluster_overview()
            
            # Send to client
            await websocket.send_json({
                "type": "cluster_overview",
                "data": overview
            })
            
            # Wait for next update
            await asyncio.sleep(5)
    except WebSocketDisconnect:
        ws_manager.disconnect(websocket, "kubernetes")
        logger.info("k8s_ws_disconnected", client=websocket.client)
    except Exception as e:
        logger.error("k8s_ws_error", error=str(e))
        ws_manager.disconnect(websocket, "kubernetes")
