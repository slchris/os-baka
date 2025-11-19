"""
USB Key and Encryption Configuration Models
Manages USB key lifecycle and multi-level encryption (TPM + USB Key + Password)
"""
from sqlalchemy import Column, Integer, String, Boolean, DateTime, Text, ForeignKey, Enum as SQLEnum
from sqlalchemy.sql import func
from sqlalchemy.orm import relationship
import enum
from app.core.database import Base


class USBKeyStatus(str, enum.Enum):
    """USB Key Status"""
    PENDING = "pending"  # 等待初始化
    ACTIVE = "active"  # 已激活可用
    BOUND = "bound"  # 已绑定到机器
    REVOKED = "revoked"  # 已吊销
    LOST = "lost"  # 丢失
    BACKUP = "backup"  # 备份状态


class EncryptionLevel(str, enum.Enum):
    """Encryption Level"""
    TPM_ONLY = "tpm_only"  # 仅TPM
    TPM_USBKEY = "tpm_usbkey"  # TPM + USB Key
    TPM_USBKEY_PASSWORD = "tpm_usbkey_password"  # TPM + USB Key + 密码
    USBKEY_ONLY = "usbkey_only"  # 仅USB Key
    USBKEY_PASSWORD = "usbkey_password"  # USB Key + 密码
    PASSWORD_ONLY = "password_only"  # 仅密码（兜底）


class USBKey(Base):
    """
    USB Key Management
    Represents a physical USB key used for encryption
    """
    __tablename__ = "usb_keys"
    
    id = Column(Integer, primary_key=True, index=True)
    
    # Key Identification
    uuid = Column(String(36), unique=True, nullable=False, index=True)  # USB Key UUID
    label = Column(String(100), nullable=False)  # 用户友好标签
    serial_number = Column(String(100), unique=True, nullable=True)  # 物理序列号
    
    # Key Material (加密存储)
    key_material = Column(Text, nullable=False)  # 主密钥材料（加密后）
    salt = Column(String(64), nullable=False)  # 盐值
    iterations = Column(Integer, default=100000)  # PBKDF2迭代次数
    
    # Backup & Recovery
    backup_key = Column(Text, nullable=True)  # 备份密钥（加密）
    recovery_code = Column(String(64), nullable=True)  # 恢复代码
    backup_count = Column(Integer, default=0)  # 备份次数
    last_backup_at = Column(DateTime(timezone=True), nullable=True)
    
    # Status
    status = Column(SQLEnum(USBKeyStatus), default=USBKeyStatus.PENDING, nullable=False)
    is_initialized = Column(Boolean, default=False, nullable=False)
    
    # Asset Binding (绑定到特定机器)
    asset_id = Column(Integer, ForeignKey("assets.id"), nullable=True, unique=True)
    asset = relationship("Asset", back_populates="usb_key")
    bound_at = Column(DateTime(timezone=True), nullable=True)
    
    # Usage Tracking
    last_used_at = Column(DateTime(timezone=True), nullable=True)
    use_count = Column(Integer, default=0)
    failed_attempts = Column(Integer, default=0)  # 失败尝试次数
    
    # Metadata
    description = Column(Text, nullable=True)
    created_by = Column(Integer, ForeignKey("users.id"), nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False)
    revoked_at = Column(DateTime(timezone=True), nullable=True)
    revoked_by = Column(Integer, ForeignKey("users.id"), nullable=True)
    
    # Relationships
    creator = relationship("User", foreign_keys=[created_by])
    revoker = relationship("User", foreign_keys=[revoked_by])
    encryption_configs = relationship("EncryptionConfig", back_populates="usb_key")
    
    def __repr__(self):
        return f"<USBKey(uuid={self.uuid}, label={self.label}, status={self.status})>"


class EncryptionConfig(Base):
    """
    Encryption Configuration for Assets
    Defines multi-level encryption strategy (TPM + USB Key + Password)
    """
    __tablename__ = "encryption_configs"
    
    id = Column(Integer, primary_key=True, index=True)
    
    # Asset Binding (一个资产一个加密配置)
    asset_id = Column(Integer, ForeignKey("assets.id"), nullable=False, unique=True)
    asset = relationship("Asset", back_populates="encryption_config")
    
    # Encryption Level
    encryption_level = Column(
        SQLEnum(EncryptionLevel), 
        default=EncryptionLevel.TPM_USBKEY_PASSWORD, 
        nullable=False
    )
    
    # TPM Configuration
    enable_tpm = Column(Boolean, default=True, nullable=False)
    tpm_version = Column(String(10), nullable=True)  # 1.2 or 2.0
    tpm_pcr_policy = Column(Text, nullable=True)  # PCR策略
    tpm_sealed_key = Column(Text, nullable=True)  # TPM封存的密钥
    
    # USB Key Configuration
    enable_usbkey = Column(Boolean, default=True, nullable=False)
    usb_key_id = Column(Integer, ForeignKey("usb_keys.id"), nullable=True)
    usb_key = relationship("USBKey", back_populates="encryption_configs")
    require_usbkey = Column(Boolean, default=True)  # 是否强制要求USB Key
    
    # Password Configuration (兜底方案)
    enable_password = Column(Boolean, default=True, nullable=False)
    password_hash = Column(String(255), nullable=True)  # LUKS密码哈希
    password_hint = Column(String(255), nullable=True)  # 密码提示
    
    # LUKS Configuration
    luks_version = Column(String(10), default="2", nullable=False)  # LUKS1 or LUKS2
    cipher = Column(String(50), default="aes-xts-plain64")
    key_size = Column(Integer, default=512)  # 密钥大小（位）
    hash_algorithm = Column(String(50), default="sha256")
    
    # Key Slots (LUKS支持多个密钥插槽)
    tpm_key_slot = Column(Integer, default=0, nullable=True)  # TPM密钥插槽
    usbkey_slot = Column(Integer, default=1, nullable=True)  # USB Key插槽
    password_slot = Column(Integer, default=2, nullable=True)  # 密码插槽
    
    # Recovery
    recovery_key = Column(Text, nullable=True)  # 恢复密钥（加密存储）
    recovery_key_slot = Column(Integer, default=7, nullable=True)  # 恢复密钥插槽
    
    # Status
    is_encrypted = Column(Boolean, default=False, nullable=False)
    encryption_initialized = Column(Boolean, default=False, nullable=False)
    last_unlocked_at = Column(DateTime(timezone=True), nullable=True)
    unlock_count = Column(Integer, default=0)
    
    # Deployment Integration
    deployed_at = Column(DateTime(timezone=True), nullable=True)
    deployment_id = Column(Integer, ForeignKey("pxe_deployments.id"), nullable=True)
    
    # Metadata
    description = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False)
    
    def __repr__(self):
        return f"<EncryptionConfig(asset_id={self.asset_id}, level={self.encryption_level})>"


class USBKeyBackup(Base):
    """
    USB Key Backup History
    Tracks backup and recovery operations
    """
    __tablename__ = "usb_key_backups"
    
    id = Column(Integer, primary_key=True, index=True)
    
    # USB Key Reference
    usb_key_id = Column(Integer, ForeignKey("usb_keys.id"), nullable=False)
    usb_key = relationship("USBKey")
    
    # Backup Data
    backup_uuid = Column(String(36), unique=True, nullable=False, index=True)
    encrypted_backup = Column(Text, nullable=False)  # 加密的备份数据
    backup_salt = Column(String(64), nullable=False)
    
    # Backup Metadata
    backup_type = Column(String(50), default="full")  # full, incremental
    backup_size = Column(Integer, nullable=True)  # 字节数
    checksum = Column(String(64), nullable=False)  # SHA256校验和
    
    # Recovery
    recovery_password = Column(String(255), nullable=True)  # 恢复密码（加密）
    is_recoverable = Column(Boolean, default=True)
    recovered_count = Column(Integer, default=0)
    last_recovered_at = Column(DateTime(timezone=True), nullable=True)
    
    # Metadata
    created_by = Column(Integer, ForeignKey("users.id"), nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    expires_at = Column(DateTime(timezone=True), nullable=True)  # 备份过期时间
    
    def __repr__(self):
        return f"<USBKeyBackup(uuid={self.backup_uuid}, usb_key_id={self.usb_key_id})>"
