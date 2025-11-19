from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from typing import List, Optional
from app.core.database import get_db
from app.core.deps import get_current_user
from app.models.asset import Asset
from app.models.user import User
from app.schemas.asset import AssetCreate, AssetUpdate, AssetResponse, AssetListResponse

router = APIRouter()


@router.get("/", response_model=AssetListResponse)
async def list_assets(
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    status: Optional[str] = Query(None, regex=r'^(active|inactive|installing|maintenance|error)$'),
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """
    List all assets with optional filtering
    
    - **skip**: Number of records to skip (pagination)
    - **limit**: Maximum number of records to return
    - **status**: Filter by asset status
    """
    query = db.query(Asset)
    
    if status:
        query = query.filter(Asset.status == status)
    
    total = query.count()
    items = query.offset(skip).limit(limit).all()
    
    return AssetListResponse(total=total, items=items)


@router.post("/", response_model=AssetResponse, status_code=201)
async def create_asset(
    asset: AssetCreate, 
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """
    Create a new asset
    
    Returns the created asset with ID and timestamps
    """
    # Check for duplicate hostname
    if db.query(Asset).filter(Asset.hostname == asset.hostname).first():
        raise HTTPException(status_code=400, detail="Hostname already exists")
    
    # Check for duplicate IP address
    if db.query(Asset).filter(Asset.ip_address == asset.ip_address).first():
        raise HTTPException(status_code=400, detail="IP address already exists")
    
    # Check for duplicate MAC address
    if db.query(Asset).filter(Asset.mac_address == asset.mac_address).first():
        raise HTTPException(status_code=400, detail="MAC address already exists")
    
    # Check for duplicate asset tag
    if db.query(Asset).filter(Asset.asset_tag == asset.asset_tag).first():
        raise HTTPException(status_code=400, detail="Asset tag already exists")
    
    db_asset = Asset(**asset.model_dump())
    db.add(db_asset)
    db.commit()
    db.refresh(db_asset)
    
    return db_asset


@router.get("/{asset_id}", response_model=AssetResponse)
async def get_asset(
    asset_id: int, 
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """
    Get a specific asset by ID
    """
    asset = db.query(Asset).filter(Asset.id == asset_id).first()
    
    if not asset:
        raise HTTPException(status_code=404, detail="Asset not found")
    
    return asset


@router.put("/{asset_id}", response_model=AssetResponse)
async def update_asset(
    asset_id: int,
    asset_update: AssetUpdate,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """
    Update an existing asset
    
    Only provided fields will be updated
    """
    db_asset = db.query(Asset).filter(Asset.id == asset_id).first()
    
    if not db_asset:
        raise HTTPException(status_code=404, detail="Asset not found")
    
    update_data = asset_update.model_dump(exclude_unset=True)
    
    # Check for duplicate values if updating unique fields
    if "hostname" in update_data:
        existing = db.query(Asset).filter(
            Asset.hostname == update_data["hostname"],
            Asset.id != asset_id
        ).first()
        if existing:
            raise HTTPException(status_code=400, detail="Hostname already exists")
    
    if "ip_address" in update_data:
        existing = db.query(Asset).filter(
            Asset.ip_address == update_data["ip_address"],
            Asset.id != asset_id
        ).first()
        if existing:
            raise HTTPException(status_code=400, detail="IP address already exists")
    
    if "mac_address" in update_data:
        existing = db.query(Asset).filter(
            Asset.mac_address == update_data["mac_address"],
            Asset.id != asset_id
        ).first()
        if existing:
            raise HTTPException(status_code=400, detail="MAC address already exists")
    
    if "asset_tag" in update_data:
        existing = db.query(Asset).filter(
            Asset.asset_tag == update_data["asset_tag"],
            Asset.id != asset_id
        ).first()
        if existing:
            raise HTTPException(status_code=400, detail="Asset tag already exists")
    
    # Update fields
    for field, value in update_data.items():
        setattr(db_asset, field, value)
    
    db.commit()
    db.refresh(db_asset)
    
    return db_asset


@router.delete("/{asset_id}", status_code=204)
async def delete_asset(
    asset_id: int, 
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """
    Delete an asset
    """
    db_asset = db.query(Asset).filter(Asset.id == asset_id).first()
    
    if not db_asset:
        raise HTTPException(status_code=404, detail="Asset not found")
    
    db.delete(db_asset)
    db.commit()
    
    return None
