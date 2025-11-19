"""
USB Key Management API Endpoints
"""
from typing import List, Optional
from fastapi import APIRouter, Depends, HTTPException, status, Response
from fastapi.responses import StreamingResponse
from sqlalchemy.orm import Session
from pydantic import BaseModel, Field
from datetime import datetime
import io

from app.core.database import get_db
from app.core.deps import get_current_active_user, get_current_superuser
from app.models.user import User
from app.models.usbkey import USBKey, USBKeyStatus, USBKeyBackup
from app.services.usbkey_service import USBKeyService


router = APIRouter()


# Schemas
class USBKeyCreate(BaseModel):
    """Create USB Key"""
    label: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = None


class USBKeyResponse(BaseModel):
    """USB Key Response"""
    id: int
    uuid: str
    label: str
    serial_number: Optional[str]
    status: str
    is_initialized: bool
    asset_id: Optional[int]
    bound_at: Optional[datetime]
    last_used_at: Optional[datetime]
    use_count: int
    failed_attempts: int
    backup_count: int
    last_backup_at: Optional[datetime]
    description: Optional[str]
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


class USBKeyInitResponse(BaseModel):
    """USB Key Initialization Response"""
    usb_key: USBKeyResponse
    key_material: str
    recovery_code: str
    uuid: str
    warning: str = "Save key_material and recovery_code securely. They will not be shown again."


class USBKeyListResponse(BaseModel):
    """USB Key List Response"""
    items: List[USBKeyResponse]
    total: int


class BindAssetRequest(BaseModel):
    """Bind USB Key to Asset"""
    asset_id: int


class CreateBackupResponse(BaseModel):
    """Backup Creation Response"""
    backup_id: int
    backup_uuid: str
    backup_password: str
    checksum: str
    created_at: datetime
    expires_at: Optional[datetime]
    warning: str = "Save backup_password securely. You'll need it for recovery."


class RestoreBackupRequest(BaseModel):
    """Restore from Backup"""
    backup_uuid: str
    backup_password: str
    new_label: Optional[str] = None


class RebuildKeyRequest(BaseModel):
    """Rebuild USB Key"""
    recovery_code: str
    new_label: Optional[str] = None


class RebuildKeyResponse(BaseModel):
    """Rebuild Key Response"""
    usb_key: USBKeyResponse
    key_material: str
    uuid: str
    warning: str = "Write this key_material to your new USB key."


class RevokeKeyRequest(BaseModel):
    """Revoke USB Key"""
    reason: Optional[str] = None


@router.get("/", response_model=USBKeyListResponse)
async def list_usb_keys(
    status: Optional[str] = None,
    asset_id: Optional[int] = None,
    skip: int = 0,
    limit: int = 100,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """List all USB keys"""
    service = USBKeyService(db)
    
    status_enum = None
    if status:
        try:
            status_enum = USBKeyStatus(status)
        except ValueError:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Invalid status: {status}"
            )
    
    keys = service.list_keys(status=status_enum, asset_id=asset_id, skip=skip, limit=limit)
    total = db.query(USBKey).count()
    
    return USBKeyListResponse(items=keys, total=total)


@router.get("/{usb_key_id}", response_model=USBKeyResponse)
async def get_usb_key(
    usb_key_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get USB key by ID"""
    usb_key = db.query(USBKey).filter(USBKey.id == usb_key_id).first()
    if not usb_key:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="USB key not found"
        )
    return usb_key


@router.post("/", response_model=USBKeyInitResponse, status_code=status.HTTP_201_CREATED)
async def initialize_usb_key(
    data: USBKeyCreate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """
    Initialize new USB key (Admin only)
    Returns key material and recovery code - save them securely!
    """
    service = USBKeyService(db)
    
    result = service.initialize_key(
        label=data.label,
        description=data.description,
        user_id=current_user.id
    )
    
    return USBKeyInitResponse(
        usb_key=result["usb_key"],
        key_material=result["key_material"],
        recovery_code=result["recovery_code"],
        uuid=result["uuid"]
    )


@router.post("/{usb_key_id}/bind", response_model=USBKeyResponse)
async def bind_to_asset(
    usb_key_id: int,
    data: BindAssetRequest,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Bind USB key to asset (Admin only)"""
    service = USBKeyService(db)
    
    try:
        usb_key = service.bind_to_asset(usb_key_id, data.asset_id)
        return usb_key
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )


@router.post("/{usb_key_id}/unbind", response_model=USBKeyResponse)
async def unbind_from_asset(
    usb_key_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Unbind USB key from asset (Admin only)"""
    service = USBKeyService(db)
    
    try:
        usb_key = service.unbind_from_asset(usb_key_id)
        return usb_key
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )


@router.post("/{usb_key_id}/backup", response_model=CreateBackupResponse)
async def create_backup(
    usb_key_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """
    Create encrypted backup of USB key (Admin only)
    Returns backup password - save it securely!
    """
    service = USBKeyService(db)
    
    try:
        result = service.create_backup(usb_key_id, user_id=current_user.id)
        
        return CreateBackupResponse(
            backup_id=result["backup"].id,
            backup_uuid=result["backup"].backup_uuid,
            backup_password=result["backup_password"],
            checksum=result["backup"].checksum,
            created_at=result["backup"].created_at,
            expires_at=result["backup"].expires_at
        )
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )


@router.get("/{usb_key_id}/backup/download")
async def download_backup(
    usb_key_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Download latest backup file (Admin only)"""
    # Get latest backup for this key
    backup = db.query(USBKeyBackup).filter(
        USBKeyBackup.usb_key_id == usb_key_id
    ).order_by(USBKeyBackup.created_at.desc()).first()
    
    if not backup:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="No backup found for this USB key"
        )
    
    # Create file stream
    backup_data = backup.encrypted_backup.encode()
    
    return StreamingResponse(
        io.BytesIO(backup_data),
        media_type="application/octet-stream",
        headers={
            "Content-Disposition": f"attachment; filename=usbkey_backup_{backup.backup_uuid}.enc"
        }
    )


@router.post("/restore", response_model=USBKeyResponse)
async def restore_from_backup(
    data: RestoreBackupRequest,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """
    Restore USB key from backup (Admin only)
    Creates a new USB key from backup
    """
    service = USBKeyService(db)
    
    try:
        usb_key = service.restore_from_backup(
            backup_uuid=data.backup_uuid,
            backup_password=data.backup_password,
            new_label=data.new_label
        )
        return usb_key
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )


@router.post("/{usb_key_id}/rebuild", response_model=RebuildKeyResponse)
async def rebuild_key(
    usb_key_id: int,
    data: RebuildKeyRequest,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """
    Rebuild USB key using recovery code (Admin only)
    Use when USB key is lost or damaged
    """
    service = USBKeyService(db)
    
    try:
        result = service.rebuild_key(
            usb_key_id=usb_key_id,
            recovery_code=data.recovery_code,
            new_label=data.new_label
        )
        
        return RebuildKeyResponse(
            usb_key=result["usb_key"],
            key_material=result["key_material"],
            uuid=result["uuid"]
        )
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )


@router.post("/{usb_key_id}/revoke", response_model=USBKeyResponse)
async def revoke_key(
    usb_key_id: int,
    data: RevokeKeyRequest,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Revoke USB key (Admin only)"""
    service = USBKeyService(db)
    
    try:
        usb_key = service.revoke_key(
            usb_key_id=usb_key_id,
            user_id=current_user.id,
            reason=data.reason
        )
        return usb_key
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )


@router.delete("/{usb_key_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_usb_key(
    usb_key_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Delete USB key (Admin only, dangerous!)"""
    usb_key = db.query(USBKey).filter(USBKey.id == usb_key_id).first()
    if not usb_key:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="USB key not found"
        )
    
    # Check if bound to asset
    if usb_key.asset_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Cannot delete USB key bound to asset. Unbind first."
        )
    
    db.delete(usb_key)
    db.commit()
    
    return None


@router.get("/{usb_key_id}/backups", response_model=List[dict])
async def list_backups(
    usb_key_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """List all backups for a USB key"""
    backups = db.query(USBKeyBackup).filter(
        USBKeyBackup.usb_key_id == usb_key_id
    ).order_by(USBKeyBackup.created_at.desc()).all()
    
    return [
        {
            "id": b.id,
            "backup_uuid": b.backup_uuid,
            "backup_size": b.backup_size,
            "checksum": b.checksum,
            "is_recoverable": b.is_recoverable,
            "recovered_count": b.recovered_count,
            "created_at": b.created_at,
            "expires_at": b.expires_at
        }
        for b in backups
    ]
