"""
PXE Service Configuration Model (Singleton)
Global configuration for PXE service with Docker container management
"""
from sqlalchemy import Column, Integer, String, Boolean, DateTime, Text, JSON
from sqlalchemy.sql import func
from app.core.database import Base


class PXEServiceConfig(Base):
    """
    Global PXE Service Configuration (Singleton)
    Only one record should exist in this table
    """
    __tablename__ = "pxe_service_config"
    
    id = Column(Integer, primary_key=True, index=True)
    
    # Network Configuration
    server_ip = Column(String(15), nullable=False, default="192.168.1.1")
    dhcp_range_start = Column(String(15), nullable=False, default="192.168.1.100")
    dhcp_range_end = Column(String(15), nullable=False, default="192.168.1.200")
    netmask = Column(String(15), nullable=False, default="255.255.255.0")
    gateway = Column(String(15), nullable=True)
    dns_servers = Column(String(255), nullable=True, default="8.8.8.8,8.8.4.4")
    
    # TFTP Configuration
    tftp_root = Column(String(255), nullable=False, default="/srv/tftp")
    
    # Boot Mode Support
    enable_bios = Column(Boolean, default=True, nullable=False)
    enable_uefi = Column(Boolean, default=True, nullable=False)
    
    # BIOS Boot Files
    bios_boot_file = Column(String(255), default="pxelinux.0")
    
    # UEFI Boot Files
    uefi_boot_file = Column(String(255), default="bootx64.efi")
    
    # OS Configuration (Debian only for now)
    os_type = Column(String(50), nullable=False, default="debian")
    debian_version = Column(String(50), nullable=False, default="12")
    debian_mirror = Column(String(255), default="http://deb.debian.org/debian")
    
    # Installation Mode
    enable_encrypted = Column(Boolean, default=True, nullable=False)  # Support encrypted installation
    enable_unencrypted = Column(Boolean, default=True, nullable=False)  # Support unencrypted installation
    default_encryption = Column(Boolean, default=False, nullable=False)  # Default to encrypted or not
    
    # Encryption Settings (for encrypted installations)
    luks_password = Column(String(255), nullable=True)  # Default LUKS password
    
    # Preseed/Kickstart Configuration
    preseed_template = Column(Text, nullable=True)  # Custom preseed template
    root_password = Column(String(255), nullable=True)  # Default root password
    default_username = Column(String(100), default="debian")
    default_user_password = Column(String(255), nullable=True)
    
    # Docker Container Configuration
    container_name = Column(String(100), default="pxe-server", nullable=False)
    container_image = Column(String(255), default="networkboot/dhcpd", nullable=False)
    container_status = Column(String(50), default="stopped")  # stopped, running, error
    container_id = Column(String(255), nullable=True)  # Docker container ID
    
    # Service Status
    service_enabled = Column(Boolean, default=False, nullable=False)
    last_started = Column(DateTime(timezone=True), nullable=True)
    last_stopped = Column(DateTime(timezone=True), nullable=True)
    
    # Additional Configuration (JSON for flexibility)
    extra_config = Column(JSON, nullable=True)
    
    # Metadata
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False)
    
    def __repr__(self):
        return f"<PXEServiceConfig(id={self.id}, enabled={self.service_enabled}, status={self.container_status})>"
