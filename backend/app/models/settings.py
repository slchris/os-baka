"""
System Settings Model
"""
from sqlalchemy import Column, Integer, String, Boolean, DateTime, Text
from sqlalchemy.sql import func
from app.core.database import Base


class SystemSetting(Base):
    """
    System configuration settings
    Key-value store for application configuration
    """
    __tablename__ = "system_settings"
    
    id = Column(Integer, primary_key=True, index=True)
    key = Column(String(255), unique=True, nullable=False, index=True)
    value = Column(Text, nullable=True)
    value_type = Column(String(50), nullable=False, default="string")  # string, int, bool, json
    category = Column(String(100), nullable=False, default="general")  # general, pxe, network, security
    description = Column(Text, nullable=True)
    is_public = Column(Boolean, default=False)  # Whether to expose to frontend
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False)
    
    def __repr__(self):
        return f"<SystemSetting {self.key}={self.value}>"
