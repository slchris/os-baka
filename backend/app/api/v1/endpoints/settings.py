"""
System Settings API Endpoints
"""
from typing import List, Optional
from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy.orm import Session
from app.core.database import get_db
from app.core.deps import get_current_active_user, get_current_superuser
from app.models.user import User
from app.models.settings import SystemSetting
from app.schemas.settings import (
    SettingCreate,
    SettingUpdate,
    SettingResponse,
    SettingListResponse
)


router = APIRouter()


@router.get("/", response_model=SettingListResponse)
async def list_settings(
    category: Optional[str] = Query(default=None),
    skip: int = Query(default=0, ge=0),
    limit: int = Query(default=100, ge=1, le=500),
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """List all settings (admin can see all, users see only public)"""
    query = db.query(SystemSetting)
    
    # Non-superusers can only see public settings
    if not current_user.is_superuser:
        query = query.filter(SystemSetting.is_public == True)
    
    if category:
        query = query.filter(SystemSetting.category == category)
    
    total = query.count()
    settings = query.offset(skip).limit(limit).all()
    
    return SettingListResponse(items=settings, total=total)


@router.get("/{key}", response_model=SettingResponse)
async def get_setting(
    key: str,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get setting by key"""
    setting = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    
    if not setting:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Setting '{key}' not found"
        )
    
    # Check permissions
    if not setting.is_public and not current_user.is_superuser:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Access denied"
        )
    
    return setting


@router.post("/", response_model=SettingResponse, status_code=status.HTTP_201_CREATED)
async def create_setting(
    setting_data: SettingCreate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Create new setting (admin only)"""
    # Check if key already exists
    existing = db.query(SystemSetting).filter(
        SystemSetting.key == setting_data.key
    ).first()
    
    if existing:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Setting '{setting_data.key}' already exists"
        )
    
    setting = SystemSetting(**setting_data.model_dump())
    db.add(setting)
    db.commit()
    db.refresh(setting)
    
    return setting


@router.put("/{key}", response_model=SettingResponse)
async def update_setting(
    key: str,
    setting_data: SettingUpdate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Update setting (admin only)"""
    setting = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    
    if not setting:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Setting '{key}' not found"
        )
    
    # Update fields
    for field, value in setting_data.model_dump(exclude_unset=True).items():
        setattr(setting, field, value)
    
    db.commit()
    db.refresh(setting)
    
    return setting


@router.delete("/{key}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_setting(
    key: str,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_superuser)
):
    """Delete setting (admin only)"""
    setting = db.query(SystemSetting).filter(SystemSetting.key == key).first()
    
    if not setting:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Setting '{key}' not found"
        )
    
    db.delete(setting)
    db.commit()
    
    return None


@router.get("/categories/list", response_model=List[str])
async def list_categories(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """List all setting categories"""
    categories = db.query(SystemSetting.category).distinct().all()
    return [cat[0] for cat in categories]
