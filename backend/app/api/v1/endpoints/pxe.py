"""
PXE Configuration API Endpoints
"""
from typing import List
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from app.core.database import get_db
from app.core.deps import get_current_active_user
from app.models.user import User
from app.models.pxe_config import PXEConfig, PXEDeployment
from app.schemas.pxe_config import (
    PXEConfigCreate,
    PXEConfigUpdate,
    PXEConfigResponse,
    PXEConfigList,
    PXEDeploymentResponse,
    DnsmasqConfigResponse,
    ServiceActionResponse
)
from app.services.pxe_service import PXEService


router = APIRouter()


@router.get("/", response_model=PXEConfigList)
async def list_pxe_configs(
    skip: int = 0,
    limit: int = 100,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """List all PXE configurations"""
    total = db.query(PXEConfig).count()
    configs = db.query(PXEConfig).offset(skip).limit(limit).all()
    
    return PXEConfigList(
        items=configs,
        total=total,
        page=skip // limit + 1 if limit > 0 else 1,
        page_size=limit
    )


@router.get("/{config_id}", response_model=PXEConfigResponse)
async def get_pxe_config(
    config_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get PXE configuration by ID"""
    config = db.query(PXEConfig).filter(PXEConfig.id == config_id).first()
    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="PXE configuration not found"
        )
    return config


@router.post("/", response_model=PXEConfigResponse, status_code=status.HTTP_201_CREATED)
async def create_pxe_config(
    config_data: PXEConfigCreate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Create new PXE configuration"""
    # Check for duplicate MAC or IP
    existing = db.query(PXEConfig).filter(
        (PXEConfig.mac_address == config_data.mac_address) |
        (PXEConfig.ip_address == config_data.ip_address) |
        (PXEConfig.hostname == config_data.hostname)
    ).first()
    
    if existing:
        if existing.mac_address == config_data.mac_address:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"MAC address {config_data.mac_address} already exists"
            )
        if existing.ip_address == config_data.ip_address:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"IP address {config_data.ip_address} already exists"
            )
        if existing.hostname == config_data.hostname:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Hostname {config_data.hostname} already exists"
            )
    
    service = PXEService(db)
    config = service.create_config(config_data)
    return config


@router.put("/{config_id}", response_model=PXEConfigResponse)
async def update_pxe_config(
    config_id: int,
    config_data: PXEConfigUpdate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Update PXE configuration"""
    service = PXEService(db)
    config = service.update_config(config_id, config_data)
    
    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="PXE configuration not found"
        )
    
    return config


@router.delete("/{config_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_pxe_config(
    config_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Delete PXE configuration"""
    service = PXEService(db)
    success = service.delete_config(config_id)
    
    if not success:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="PXE configuration not found"
        )
    
    return None


@router.post("/generate-all", response_model=dict)
async def generate_all_configs(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Generate dnsmasq configurations for all enabled PXE configs"""
    service = PXEService(db)
    generated = service.generate_all_configs()
    
    return {
        "success": True,
        "message": f"Generated {len(generated)} configurations",
        "details": generated
    }


@router.post("/validate", response_model=DnsmasqConfigResponse)
async def validate_dnsmasq_config(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Validate dnsmasq configuration"""
    service = PXEService(db)
    validation = service.validate_dnsmasq_config()
    
    return DnsmasqConfigResponse(
        config_content=validation.get('output', ''),
        config_path='/etc/dnsmasq.conf',
        valid=validation['valid'],
        errors=validation['errors']
    )


@router.post("/restart-service", response_model=ServiceActionResponse)
async def restart_dnsmasq_service(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Restart dnsmasq service"""
    if not current_user.is_superuser:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Only administrators can restart services"
        )
    
    service = PXEService(db)
    result = service.restart_dnsmasq()
    
    return ServiceActionResponse(**result)


@router.get("/service-status", response_model=dict)
async def get_service_status(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get dnsmasq service status"""
    service = PXEService(db)
    status_info = service.get_dnsmasq_status()
    return status_info


@router.get("/{config_id}/deployments", response_model=List[PXEDeploymentResponse])
async def list_deployments(
    config_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """List deployments for a PXE configuration"""
    deployments = db.query(PXEDeployment).filter(
        PXEDeployment.pxe_config_id == config_id
    ).order_by(PXEDeployment.started_at.desc()).all()
    
    return deployments


@router.post("/{config_id}/deploy", response_model=PXEDeploymentResponse)
async def create_deployment(
    config_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Create a deployment for a PXE configuration"""
    # Check if config exists
    config = db.query(PXEConfig).filter(PXEConfig.id == config_id).first()
    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="PXE configuration not found"
        )
    
    service = PXEService(db)
    deployment = service.create_deployment(config_id, current_user.id)
    
    return deployment
