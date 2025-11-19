"""
PXE Configuration Schemas
"""
from typing import Optional
from datetime import datetime
from pydantic import BaseModel, Field, field_validator


class PXEConfigBase(BaseModel):
    """Base PXE configuration schema"""
    hostname: str = Field(..., min_length=1, max_length=255)
    mac_address: str = Field(..., pattern=r'^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$')
    ip_address: str = Field(..., pattern=r'^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$')
    netmask: str = Field(default="255.255.255.0", pattern=r'^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$')
    gateway: Optional[str] = Field(None, pattern=r'^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$')
    dns_servers: Optional[str] = None
    boot_image: Optional[str] = None
    boot_params: Optional[str] = None
    os_type: str = Field(default="ubuntu")
    enabled: bool = Field(default=True)
    description: Optional[str] = None
    asset_id: Optional[int] = None
    
    @field_validator('mac_address')
    @classmethod
    def normalize_mac(cls, v: str) -> str:
        """Normalize MAC address to uppercase with colons"""
        return v.upper().replace('-', ':')


class PXEConfigCreate(PXEConfigBase):
    """Schema for creating PXE configuration"""
    pass


class PXEConfigUpdate(BaseModel):
    """Schema for updating PXE configuration"""
    hostname: Optional[str] = None
    mac_address: Optional[str] = None
    ip_address: Optional[str] = None
    netmask: Optional[str] = None
    gateway: Optional[str] = None
    dns_servers: Optional[str] = None
    boot_image: Optional[str] = None
    boot_params: Optional[str] = None
    os_type: Optional[str] = None
    enabled: Optional[bool] = None
    description: Optional[str] = None
    asset_id: Optional[int] = None


class PXEConfigResponse(PXEConfigBase):
    """Schema for PXE configuration response"""
    id: int
    deployed: bool
    last_boot: Optional[datetime]
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


class PXEConfigList(BaseModel):
    """Schema for paginated PXE configuration list"""
    items: list[PXEConfigResponse]
    total: int
    page: int
    page_size: int


class PXEDeploymentBase(BaseModel):
    """Base deployment schema"""
    pxe_config_id: int
    

class PXEDeploymentCreate(PXEDeploymentBase):
    """Schema for creating deployment"""
    pass


class PXEDeploymentResponse(PXEDeploymentBase):
    """Schema for deployment response"""
    id: int
    started_at: datetime
    completed_at: Optional[datetime]
    status: str
    log_output: Optional[str]
    error_message: Optional[str]
    initiated_by: Optional[int]
    
    class Config:
        from_attributes = True


class DnsmasqConfigResponse(BaseModel):
    """Response for dnsmasq configuration"""
    config_content: str
    config_path: str
    valid: bool
    errors: list[str] = []


class ServiceActionResponse(BaseModel):
    """Response for service actions (restart, reload, etc.)"""
    service: str
    action: str
    success: bool
    message: str
    output: Optional[str] = None
