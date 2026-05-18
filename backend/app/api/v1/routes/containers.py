import re

from fastapi import APIRouter, Depends, HTTPException, Query

from app.core.logging import get_logger
from app.domain.entities.user import User
from app.domain.services.docker_service import DockerService
from app.schemas.container import (
    ContainerActionResponse,
    ContainerDetailResponse,
    ContainerResponse,
    LogLine,
)
from app.utils.deps import (
    get_audit_repo,
    get_container_repo,
    get_current_user,
    get_docker,
    require_admin,
)

router = APIRouter()
logger = get_logger("api.containers")

# Docker container IDs are 12 or 64 hex chars; names are alphanumeric with underscores/dashes/dots
_CONTAINER_ID_RE = re.compile(r"^[a-zA-Z0-9][a-zA-Z0-9_.\-]{1,127}$")


def _validate_container_id(container_id: str) -> str:
    if not _CONTAINER_ID_RE.match(container_id):
        raise HTTPException(status_code=400, detail="Invalid container ID or name.")
    return container_id


@router.get("")
async def list_containers(
    all: bool = Query(False),
    current_user: User = Depends(get_current_user),
    docker=Depends(get_docker),
    audit_repo=Depends(get_audit_repo),
):
    logger.info("containers_list", user_id=str(current_user.id), all=all)
    try:
        service = DockerService(docker, None, audit_repo)
        containers = await service.list_containers(all=all)
        logger.info("containers_list_ok", count=len(containers) if isinstance(containers, list) else "?")
        return containers
    except Exception as e:
        logger.error("containers_list_error", error=str(e), exc_info=True)
        raise HTTPException(status_code=503, detail="Docker is unavailable or returned an error")


@router.get("/{container_id}")
async def inspect_container(
    container_id: str,
    current_user: User = Depends(get_current_user),
    docker=Depends(get_docker),
    audit_repo=Depends(get_audit_repo),
):
    _validate_container_id(container_id)
    logger.info("container_inspect", container_id=container_id, user_id=str(current_user.id))
    try:
        service = DockerService(docker, None, audit_repo)
        info = await service.inspect_container(container_id)
        return info
    except ValueError as e:
        logger.warning("container_not_found", container_id=container_id, error=str(e))
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error("container_inspect_error", container_id=container_id, error=str(e), exc_info=True)
        raise HTTPException(status_code=503, detail="Failed to inspect container")


@router.post("/{container_id}/start", response_model=ContainerActionResponse)
async def start_container(
    container_id: str,
    current_user: User = Depends(require_admin),
    docker=Depends(get_docker),
    audit_repo=Depends(get_audit_repo),
):
    _validate_container_id(container_id)
    logger.info("container_start", container_id=container_id, user_id=str(current_user.id))
    try:
        service = DockerService(docker, None, audit_repo)
        result = await service.start_container(container_id, user_id=current_user.id)
        logger.info("container_start_ok", container_id=container_id)
        return ContainerActionResponse(**result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error("container_start_error", container_id=container_id, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/{container_id}/stop", response_model=ContainerActionResponse)
async def stop_container(
    container_id: str,
    current_user: User = Depends(require_admin),
    docker=Depends(get_docker),
    audit_repo=Depends(get_audit_repo),
):
    _validate_container_id(container_id)
    logger.info("container_stop", container_id=container_id, user_id=str(current_user.id))
    try:
        service = DockerService(docker, None, audit_repo)
        result = await service.stop_container(container_id, user_id=current_user.id)
        logger.info("container_stop_ok", container_id=container_id)
        return ContainerActionResponse(**result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error("container_stop_error", container_id=container_id, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/{container_id}/restart", response_model=ContainerActionResponse)
async def restart_container(
    container_id: str,
    current_user: User = Depends(require_admin),
    docker=Depends(get_docker),
    audit_repo=Depends(get_audit_repo),
):
    _validate_container_id(container_id)
    logger.info("container_restart", container_id=container_id, user_id=str(current_user.id))
    try:
        service = DockerService(docker, None, audit_repo)
        result = await service.restart_container(container_id, user_id=current_user.id)
        logger.info("container_restart_ok", container_id=container_id)
        return ContainerActionResponse(**result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error("container_restart_error", container_id=container_id, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/{container_id}/logs")
async def get_container_logs(
    container_id: str,
    tail: int = Query(100, ge=1, le=1000),
    since: str | None = Query(None),
    current_user: User = Depends(get_current_user),
    docker=Depends(get_docker),
    audit_repo=Depends(get_audit_repo),
):
    _validate_container_id(container_id)
    logger.info("container_logs_fetch", container_id=container_id, tail=tail, user_id=str(current_user.id))
    try:
        service = DockerService(docker, None, audit_repo)
        logs = await service.container_logs(container_id, tail=tail, since=since)
        return {"container_id": container_id, "logs": logs}
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error("container_logs_error", container_id=container_id, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))
