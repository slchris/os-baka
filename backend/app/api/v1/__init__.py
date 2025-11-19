from fastapi import APIRouter
from app.api.v1.endpoints import assets, auth, pxe, pxe_service, deployments, settings, usbkeys

api_router = APIRouter()

api_router.include_router(auth.router, prefix="/auth", tags=["authentication"])
api_router.include_router(assets.router, prefix="/assets", tags=["assets"])
api_router.include_router(pxe.router, prefix="/pxe", tags=["pxe"])  # Legacy PXE configs (deprecated)
api_router.include_router(pxe_service.router, prefix="/pxe-service", tags=["pxe-service"])  # New PXE service
api_router.include_router(deployments.router, prefix="/deployments", tags=["deployments"])
api_router.include_router(settings.router, prefix="/settings", tags=["settings"])
api_router.include_router(usbkeys.router, prefix="/usbkeys", tags=["usbkeys"])
