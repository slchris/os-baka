"""
PXE Configuration Model
Manages PXE boot configurations for network-based OS deployment
"""
from sqlalchemy import Column, Integer, String, Boolean, DateTime, Text, ForeignKey
from sqlalchemy.sql import func
from sqlalchemy.orm import relationship
from app.core.database import Base


class PXEConfig(Base):
    """
    PXE Configuration for network booting
    Links MAC address to IP assignment and boot configuration
    """
    __tablename__ = "pxe_configs"
    
    id = Column(Integer, primary_key=True, index=True)
    hostname = Column(String(255), unique=True, nullable=False, index=True)
    mac_address = Column(String(17), unique=True, nullable=False, index=True)
    ip_address = Column(String(15), unique=True, nullable=False)
    netmask = Column(String(15), nullable=False, default="255.255.255.0")
    gateway = Column(String(15), nullable=True)
    dns_servers = Column(String(255), nullable=True)  # Comma-separated DNS servers
    
    # Boot configuration
    boot_image = Column(String(255), nullable=True)  # Path to boot image
    boot_params = Column(Text, nullable=True)  # Additional boot parameters
    os_type = Column(String(50), nullable=False, default="ubuntu")
    
    # PXE status
    enabled = Column(Boolean, default=True)
    deployed = Column(Boolean, default=False)
    last_boot = Column(DateTime(timezone=True), nullable=True)
    
    # Metadata
    description = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False)
    
    # Relationship to asset (optional) - DEPRECATED: use Asset model directly
    asset_id = Column(Integer, ForeignKey("assets.id"), nullable=True)
    asset = relationship("Asset")


class PXEDeployment(Base):
    """
    Track PXE deployment history
    """
    __tablename__ = "pxe_deployments"
    
    id = Column(Integer, primary_key=True, index=True)
    pxe_config_id = Column(Integer, ForeignKey("pxe_configs.id"), nullable=False)
    pxe_config = relationship("PXEConfig")
    
    # Deployment details
    started_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    completed_at = Column(DateTime(timezone=True), nullable=True)
    status = Column(String(50), nullable=False, default="pending")  # pending, in_progress, completed, failed
    
    # Logs
    log_output = Column(Text, nullable=True)
    error_message = Column(Text, nullable=True)
    
    # User tracking
    initiated_by = Column(Integer, ForeignKey("users.id"), nullable=True)
    user = relationship("User")
