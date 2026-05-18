from fastapi import APIRouter, Depends, HTTPException

from app.core.logging import get_logger
from app.domain.services.dashboard_service import DashboardService
from app.schemas.dashboard import DashboardResponse
from app.domain.entities.user import User
from app.utils.deps import (
    get_container_event_repo,
    get_container_repo,
    get_current_user,
    get_docker,
    get_host_repo,
    get_metrics_repo,
    get_stack_repo,
)

router = APIRouter()
logger = get_logger("api.dashboard")


@router.get("", response_model=DashboardResponse)
async def get_dashboard(
    current_user: User = Depends(get_current_user),
    docker=Depends(get_docker),
    host_repo=Depends(get_host_repo),
    metrics_repo=Depends(get_metrics_repo),
    container_repo=Depends(get_container_repo),
    event_repo=Depends(get_container_event_repo),
    stack_repo=Depends(get_stack_repo),
):
    logger.info("dashboard_fetch", user_id=str(current_user.id))
    try:
        service = DashboardService(docker, host_repo, metrics_repo, container_repo, event_repo, stack_repo)
        data = await service.get_dashboard()
        logger.info("dashboard_ok", user_id=str(current_user.id))
        return DashboardResponse(**data)
    except Exception as e:
        logger.error("dashboard_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to load dashboard")
