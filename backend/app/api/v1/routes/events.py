from fastapi import APIRouter, Depends, HTTPException, Query

from app.core.logging import get_logger
from app.domain.entities.user import User
from app.utils.deps import get_container_event_repo, get_current_user

router = APIRouter()
logger = get_logger("api.events")


@router.get("")
async def get_events(
    limit: int = Query(50, ge=1, le=200),
    current_user: User = Depends(get_current_user),
    event_repo=Depends(get_container_event_repo),
):
    logger.info("events_fetch", limit=limit, user_id=str(current_user.id))
    try:
        events = await event_repo.get_recent(limit=limit)
        return [
            {
                "id": str(e.id),
                "container_docker_id": e.container_docker_id,
                "container_name": e.container_name,
                "event_type": e.event_type.value if hasattr(e.event_type, "value") else str(e.event_type),
                "timestamp": e.timestamp.isoformat() if e.timestamp else None,
                "metadata": e.metadata or {},
            }
            for e in events
        ]
    except Exception as e:
        logger.error("events_fetch_error", limit=limit, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to load events")
