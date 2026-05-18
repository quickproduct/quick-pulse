from fastapi import APIRouter, Depends, HTTPException, Query

from app.core.logging import get_logger
from app.domain.entities.user import User
from app.domain.services.metrics_service import MetricsService
from app.schemas.metrics import MetricHistoryResponse, MetricSnapshot
from app.utils.deps import (
    get_current_user,
    get_host_repo,
    get_metrics_repo,
)

router = APIRouter()
logger = get_logger("api.metrics")

_VALID_METRICS = {"cpu", "memory", "disk", "network", "load"}
_VALID_RANGES = {"15m", "1h", "6h", "24h", "7d"}


@router.get("/summary")
async def get_metrics_summary(
    current_user: User = Depends(get_current_user),
    metrics_repo=Depends(get_metrics_repo),
    host_repo=Depends(get_host_repo),
):
    logger.info("metrics_summary_fetch", user_id=str(current_user.id))
    try:
        service = MetricsService(metrics_repo, host_repo)
        data = await service.get_summary()
        return data
    except Exception as e:
        logger.error("metrics_summary_error", user_id=str(current_user.id), error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to load metrics summary")


@router.get("/history", response_model=MetricHistoryResponse)
async def get_metrics_history(
    metric: str = Query("cpu"),
    range: str = Query("1h"),
    current_user: User = Depends(get_current_user),
    metrics_repo=Depends(get_metrics_repo),
    host_repo=Depends(get_host_repo),
):
    if metric not in _VALID_METRICS:
        raise HTTPException(status_code=400, detail=f"Invalid metric '{metric}'. Valid: {sorted(_VALID_METRICS)}")
    if range not in _VALID_RANGES:
        raise HTTPException(status_code=400, detail=f"Invalid range '{range}'. Valid: {sorted(_VALID_RANGES)}")

    logger.info("metrics_history_fetch", metric=metric, range=range, user_id=str(current_user.id))
    try:
        service = MetricsService(metrics_repo, host_repo)
        data = await service.get_history(metric, range)
        return MetricHistoryResponse(**data)
    except Exception as e:
        logger.error("metrics_history_error", metric=metric, range=range, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to load metrics history")
