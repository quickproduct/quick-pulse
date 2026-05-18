from fastapi import APIRouter, Depends, HTTPException

from app.core.logging import get_logger
from app.domain.entities.user import User
from app.domain.services.compose_service import ComposeService as ComposeSvc
from app.schemas.stack import StackActionResponse
from app.utils.deps import (
    get_audit_repo,
    get_current_user,
    get_docker,
    get_stack_repo,
    require_admin,
)

router = APIRouter()
logger = get_logger("api.stacks")


@router.get("")
async def list_stacks(
    current_user: User = Depends(get_current_user),
    docker=Depends(get_docker),
    stack_repo=Depends(get_stack_repo),
    audit_repo=Depends(get_audit_repo),
):
    logger.info("stacks_list", user_id=str(current_user.id))
    try:
        service = ComposeSvc(docker, stack_repo, audit_repo)
        stacks = await service.list_stacks()
        logger.info("stacks_list_ok", count=len(stacks) if isinstance(stacks, list) else "?")
        return stacks
    except Exception as e:
        logger.error("stacks_list_error", error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to list stacks")


@router.get("/{name}")
async def get_stack(
    name: str,
    current_user: User = Depends(get_current_user),
    docker=Depends(get_docker),
    stack_repo=Depends(get_stack_repo),
    audit_repo=Depends(get_audit_repo),
):
    logger.info("stack_get", name=name, user_id=str(current_user.id))
    try:
        service = ComposeSvc(docker, stack_repo, audit_repo)
        stack = await service.get_stack(name)
        if not stack:
            logger.warning("stack_not_found", name=name)
            raise HTTPException(status_code=404, detail="Stack not found")
        return stack
    except HTTPException:
        raise
    except Exception as e:
        logger.error("stack_get_error", name=name, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail="Failed to get stack")


@router.post("/{name}/start", response_model=StackActionResponse)
async def start_stack(
    name: str,
    current_user: User = Depends(require_admin),
    docker=Depends(get_docker),
    stack_repo=Depends(get_stack_repo),
    audit_repo=Depends(get_audit_repo),
):
    logger.info("stack_start", name=name, user_id=str(current_user.id))
    try:
        service = ComposeSvc(docker, stack_repo, audit_repo)
        result = await service.start_stack(name, user_id=current_user.id)
        logger.info("stack_start_ok", name=name)
        return StackActionResponse(**result)
    except Exception as e:
        logger.error("stack_start_error", name=name, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/{name}/stop", response_model=StackActionResponse)
async def stop_stack(
    name: str,
    current_user: User = Depends(require_admin),
    docker=Depends(get_docker),
    stack_repo=Depends(get_stack_repo),
    audit_repo=Depends(get_audit_repo),
):
    logger.info("stack_stop", name=name, user_id=str(current_user.id))
    try:
        service = ComposeSvc(docker, stack_repo, audit_repo)
        result = await service.stop_stack(name, user_id=current_user.id)
        logger.info("stack_stop_ok", name=name)
        return StackActionResponse(**result)
    except Exception as e:
        logger.error("stack_stop_error", name=name, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/{name}/restart", response_model=StackActionResponse)
async def restart_stack(
    name: str,
    current_user: User = Depends(require_admin),
    docker=Depends(get_docker),
    stack_repo=Depends(get_stack_repo),
    audit_repo=Depends(get_audit_repo),
):
    logger.info("stack_restart", name=name, user_id=str(current_user.id))
    try:
        service = ComposeSvc(docker, stack_repo, audit_repo)
        result = await service.restart_stack(name, user_id=current_user.id)
        logger.info("stack_restart_ok", name=name)
        return StackActionResponse(**result)
    except Exception as e:
        logger.error("stack_restart_error", name=name, error=str(e), exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))
