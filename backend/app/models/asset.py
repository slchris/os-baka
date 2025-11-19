from sqlalchemy import Column, Integer, String, DateTime, Boolean
from sqlalchemy.sql import func
from sqlalchemy.orm import relationship
from app.core.database import Base


class Asset(Base):
    """Asset model for physical device management"""
    
    __tablename__ = "assets"
    
    id = Column(Integer, primary_key=True, index=True)
    hostname = Column(String(255), unique=True, index=True, nullable=False)
    ip_address = Column(String(45), unique=True, index=True, nullable=False)  # Support IPv6
    mac_address = Column(String(17), unique=True, index=True, nullable=False)
    asset_tag = Column(String(100), unique=True, index=True, nullable=False)
    
    # Status: active, inactive, installing, maintenance, error
    status = Column(String(50), default="inactive", nullable=False)
    
    # Additional metadata
    os_type = Column(String(50), nullable=True)  # macos, linux, windows
    encryption_enabled = Column(Boolean, default=False)
    last_seen = Column(DateTime(timezone=True), nullable=True)
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), onupdate=func.now())
    
    # Relationships
    usb_key = relationship("USBKey", back_populates="asset", uselist=False)
    encryption_config = relationship("EncryptionConfig", back_populates="asset", uselist=False)
    
    def __repr__(self):
        return f"<Asset {self.hostname} ({self.ip_address})>"
