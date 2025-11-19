"""
PXE Service API Endpoints (Singleton Configuration)
"""
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from app.core.database import get_db
from app.core.deps import get_current_active_user, get_current_superuser
from app.models.user import User
from app.services.pxe_docker_service import PXEDockerService
from pydantic import BaseModel, Field
from typing import Optional, Dict, Any
from datetime import datetime


router = APIRouter()


# Schemas
class PXEServiceConfigResponse(BaseModel):
    """PXE Service Configuration Response"""
    id: int
    
    # Network Configuration
    server_ip: str
    dhcp_range_start: str
    dhcp_range_end: str
    netmask: str
    gateway: Optional[str]
    dns_servers: Optional[str]
    
    # TFTP Configuration
    tftp_root: str
    
    # Boot Mode Support
    enable_bios: bool
    enable_uefi: bool
    bios_boot_file: str
    uefi_boot_file: str
    
    # OS Configuration
    os_type: str
    debian_version: str
    debian_mirror: str
    
    # Installation Mode
    enable_encrypted: bool
    enable_unencrypted: bool
    default_encryption: bool
    
    # Preseed Configuration
    default_username: str
    
    # Docker Container
    container_name: str
    container_image: str
    container_status: str
    container_id: Optional[str]
    
    # Service Status
    service_enabled: bool
    last_started: Optional[datetime]
    last_stopped: Optional[datetime]
    
    # Metadata
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


class PXEServiceConfigUpdate(BaseModel):
    """PXE Service Configuration Update"""
    # Network Configuration
    server_ip: Optional[str] = None
    dhcp_range_start: Optional[str] = None
    dhcp_range_end: Optional[str] = None
    netmask: Optional[str] = None
    gateway: Optional[str] = None
    dns_servers: Optional[str] = None
    
    # TFTP Configuration
    tftp_root: Optional[str] = None
    
    # Boot Mode Support
    enable_bios: Optional[bool] = None
    enable_uefi: Optional[bool] = None
    bios_boot_file: Optional[str] = None
    uefi_boot_file: Optional[str] = None
    
    # OS Configuration
    debian_version: Optional[str] = None
    debian_mirror: Optional[str] = None
    
    # Installation Mode
    enable_encrypted: Optional[bool] = None
    enable_unencrypted: Optional[bool] = None
    default_encryption: Optional[bool] = None
    
    # Encryption Settings
    luks_password: Optional[str] = None
    
    # Preseed Configuration
    preseed_template: Optional[str] = None
    root_password: Optional[str] = None
    default_username: Optional[str] = None
    default_user_password: Optional[str] = None
    
    # Docker Container
    container_image: Optional[str] = None


class ServiceStatusResponse(BaseModel):
    """Service Status Response"""
    status: str
    container_exists: bool
    container_running: bool
    enabled: bool
    last_started: Optional[str]
    last_stopped: Optional[str]


class ServiceActionResponse(BaseModel):
    """Service Action Response"""
    success: bool
    message: str
    status: str


class LogsResponse(BaseModel):
    """Logs Response"""
    success: bool
    logs: str
    message: Optional[str] = None


@router.get("/config", response_model=PXEServiceConfigResponse)
async def get_pxe_config(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get PXE service configuration (singleton)"""
    service = PXEDockerService(db)
    config = service.get_config()
    return config


@router.put("/config", response_model=PXEServiceConfigResponse)
async def update_pxe_config(
    config_data: PXEServiceConfigUpdate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Update PXE service configuration (admin only)"""
    service = PXEDockerService(db)
    
    # Extract non-None values
    update_data = {k: v for k, v in config_data.dict().items() if v is not None}
    
    if not update_data:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="No fields to update"
        )
    
    config = service.update_config(**update_data)
    return config


@router.get("/status", response_model=ServiceStatusResponse)
async def get_service_status(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get PXE service status"""
    service = PXEDockerService(db)
    status_info = service.get_status()
    return status_info


@router.post("/start", response_model=ServiceActionResponse)
async def start_service(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Start PXE Docker service (admin only)"""
    service = PXEDockerService(db)
    result = service.start_service()
    
    if not result["success"]:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=result["message"]
        )
    
    return result


@router.post("/stop", response_model=ServiceActionResponse)
async def stop_service(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Stop PXE Docker service (admin only)"""
    service = PXEDockerService(db)
    result = service.stop_service()
    
    if not result["success"]:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=result["message"]
        )
    
    return result


@router.post("/restart", response_model=ServiceActionResponse)
async def restart_service(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Restart PXE Docker service (admin only)"""
    service = PXEDockerService(db)
    result = service.restart_service()
    
    if not result["success"]:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=result["message"]
        )
    
    return result


@router.get("/logs", response_model=LogsResponse)
async def get_service_logs(
    tail: int = 100,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get PXE service logs"""
    service = PXEDockerService(db)
    result = service.get_logs(tail=tail)
    
    return result


@router.delete("/container", response_model=ServiceActionResponse)
async def remove_container(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Remove PXE Docker container (admin only, must be stopped first)"""
    service = PXEDockerService(db)
    result = service.remove_container()
    
    if not result["success"]:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=result["message"]
        )
    
    return result
