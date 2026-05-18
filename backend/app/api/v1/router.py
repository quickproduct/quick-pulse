from fastapi import APIRouter

from app.api.v1.routes import auth, billing, dashboard, containers, organizations, stacks, metrics, events, alerts, kubernetes

router = APIRouter(prefix="/api/v1")

router.include_router(auth.router, prefix="/auth", tags=["auth"])
# /me alias at top level — frontend calls /api/v1/me directly
router.add_api_route("/me", auth.me, methods=["GET"], tags=["auth"])
router.include_router(dashboard.router, prefix="/dashboard", tags=["dashboard"])
router.include_router(containers.router, prefix="/containers", tags=["containers"])
router.include_router(stacks.router, prefix="/stacks", tags=["stacks"])
router.include_router(metrics.router, prefix="/metrics", tags=["metrics"])
router.include_router(events.router, prefix="/events", tags=["events"])
router.include_router(alerts.router, prefix="/alerts", tags=["alerts"])
router.include_router(alerts.rules_router, prefix="/alert-rules", tags=["alert-rules"])
router.include_router(organizations.router, prefix="/organizations", tags=["organizations"])
router.include_router(billing.router, prefix="/billing", tags=["billing"])
router.include_router(kubernetes.router, prefix="/kubernetes", tags=["kubernetes"])
