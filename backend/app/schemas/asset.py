from pydantic import BaseModel, Field, validator
from datetime import datetime
from typing import Optional
import re


class AssetBase(BaseModel):
    """Base schema for Asset"""
    hostname: str = Field(..., min_length=1, max_length=255)
    ip_address: str = Field(..., pattern=r'^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$')
    mac_address: str = Field(..., pattern=r'^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$')
    asset_tag: str = Field(..., min_length=1, max_length=100)
    usb_key: Optional[str] = Field(None, max_length=255)
    status: str = Field(default="inactive", pattern=r'^(active|inactive|installing|maintenance|error)$')
    os_type: Optional[str] = Field(None, pattern=r'^(macos|linux|windows)$')
    encryption_enabled: bool = False
    
    @validator('mac_address')
    def normalize_mac_address(cls, v):
        """Normalize MAC address to lowercase with colons"""
        v = v.lower().replace('-', ':')
        return v


class AssetCreate(AssetBase):
    """Schema for creating a new asset"""
    pass


class AssetUpdate(BaseModel):
    """Schema for updating an asset"""
    hostname: Optional[str] = Field(None, min_length=1, max_length=255)
    ip_address: Optional[str] = Field(None, pattern=r'^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$')
    mac_address: Optional[str] = Field(None, pattern=r'^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$')
    asset_tag: Optional[str] = Field(None, min_length=1, max_length=100)
    usb_key: Optional[str] = Field(None, max_length=255)
    status: Optional[str] = Field(None, pattern=r'^(active|inactive|installing|maintenance|error)$')
    os_type: Optional[str] = Field(None, pattern=r'^(macos|linux|windows)$')
    encryption_enabled: Optional[bool] = None


class AssetResponse(AssetBase):
    """Schema for asset response"""
    id: int
    last_seen: Optional[datetime] = None
    created_at: datetime
    updated_at: Optional[datetime] = None
    
    class Config:
        from_attributes = True


class AssetListResponse(BaseModel):
    """Schema for asset list response"""
    total: int
    items: list[AssetResponse]
