"""
System Settings Schemas
"""
from typing import Optional
from datetime import datetime
from pydantic import BaseModel, Field


class SettingBase(BaseModel):
    """Base setting schema"""
    key: str = Field(..., min_length=1, max_length=255)
    value: Optional[str] = None
    value_type: str = Field(default="string")
    category: str = Field(default="general")
    description: Optional[str] = None
    is_public: bool = Field(default=False)


class SettingCreate(SettingBase):
    """Schema for creating setting"""
    pass


class SettingUpdate(BaseModel):
    """Schema for updating setting"""
    value: Optional[str] = None
    description: Optional[str] = None
    is_public: Optional[bool] = None


class SettingResponse(SettingBase):
    """Schema for setting response"""
    id: int
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


class SettingListResponse(BaseModel):
    """Schema for settings list"""
    items: list[SettingResponse]
    total: int
