"""
Deployment Management API Endpoints
"""
from typing import List, Optional
from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy.orm import Session
from sqlalchemy import desc
from app.core.database import get_db
from app.core.deps import get_current_active_user
from app.models.user import User
from app.models.pxe_config import PXEDeployment
from app.schemas.pxe_config import PXEDeploymentResponse


router = APIRouter()


@router.get("/", response_model=List[PXEDeploymentResponse])
async def list_deployments(
    skip: int = Query(default=0, ge=0),
    limit: int = Query(default=50, ge=1, le=100),
    status_filter: Optional[str] = Query(default=None, alias="status"),
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """List all deployments with optional filtering"""
    query = db.query(PXEDeployment)
    
    if status_filter:
        query = query.filter(PXEDeployment.status == status_filter)
    
    deployments = query.order_by(
        desc(PXEDeployment.started_at)
    ).offset(skip).limit(limit).all()
    
    return deployments


@router.get("/{deployment_id}", response_model=PXEDeploymentResponse)
async def get_deployment(
    deployment_id: int,
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get deployment by ID"""
    deployment = db.query(PXEDeployment).filter(
        PXEDeployment.id == deployment_id
    ).first()
    
    if not deployment:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Deployment not found"
        )
    
    return deployment


@router.get("/stats/summary", response_model=dict)
async def get_deployment_stats(
    db: Session = Depends(get_db),
    current_user: User = Depends(get_current_active_user)
):
    """Get deployment statistics"""
    total = db.query(PXEDeployment).count()
    
    pending = db.query(PXEDeployment).filter(
        PXEDeployment.status == 'pending'
    ).count()
    
    in_progress = db.query(PXEDeployment).filter(
        PXEDeployment.status == 'in_progress'
    ).count()
    
    completed = db.query(PXEDeployment).filter(
        PXEDeployment.status == 'completed'
    ).count()
    
    failed = db.query(PXEDeployment).filter(
        PXEDeployment.status == 'failed'
    ).count()
    
    return {
        "total": total,
        "pending": pending,
        "in_progress": in_progress,
        "completed": completed,
        "failed": failed
    }
