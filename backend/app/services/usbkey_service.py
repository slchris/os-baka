"""
USB Key Service
Manages USB key lifecycle, encryption, backup, and recovery
"""
import os
import uuid
import hashlib
import secrets
import base64
from datetime import datetime, timedelta
from typing import Optional, Dict, List
from sqlalchemy.orm import Session
from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.backends import default_backend

from app.models.usbkey import USBKey, USBKeyStatus, EncryptionConfig, EncryptionLevel, USBKeyBackup
from app.models.asset import Asset


class USBKeyService:
    """USB Key Management Service"""
    
    def __init__(self, db: Session):
        self.db = db
        self.master_key = os.getenv("USBKEY_MASTER_KEY", "change-me-in-production").encode()
    
    def _derive_key(self, password: str, salt: bytes, iterations: int = 100000) -> bytes:
        """Derive encryption key from password using PBKDF2"""
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=salt,
            iterations=iterations,
            backend=default_backend()
        )
        return base64.urlsafe_b64encode(kdf.derive(password.encode()))
    
    def _encrypt_data(self, data: str, key: bytes) -> str:
        """Encrypt data using Fernet"""
        f = Fernet(key)
        return f.encrypt(data.encode()).decode()
    
    def _decrypt_data(self, encrypted_data: str, key: bytes) -> str:
        """Decrypt data using Fernet"""
        f = Fernet(key)
        return f.decrypt(encrypted_data.encode()).decode()
    
    def initialize_key(
        self,
        label: str,
        description: Optional[str] = None,
        user_id: Optional[int] = None
    ) -> USBKey:
        """
        Initialize a new USB key
        生成密钥材料、UUID、恢复代码
        """
        # Generate key material
        key_material = secrets.token_hex(32)  # 256-bit key
        key_uuid = str(uuid.uuid4())
        recovery_code = secrets.token_urlsafe(32)
        
        # Generate salt
        salt = secrets.token_hex(32)
        
        # Encrypt key material with master key
        encryption_key = self._derive_key(self.master_key.decode(), bytes.fromhex(salt))
        encrypted_key_material = self._encrypt_data(key_material, encryption_key)
        
        # Create USB key record
        usb_key = USBKey(
            uuid=key_uuid,
            label=label,
            key_material=encrypted_key_material,
            salt=salt,
            recovery_code=hashlib.sha256(recovery_code.encode()).hexdigest(),
            status=USBKeyStatus.ACTIVE,
            is_initialized=True,
            description=description,
            created_by=user_id
        )
        
        self.db.add(usb_key)
        self.db.commit()
        self.db.refresh(usb_key)
        
        # Return key material and recovery code (only shown once)
        return {
            "usb_key": usb_key,
            "key_material": key_material,  # Save this to USB key
            "recovery_code": recovery_code,  # Save this securely
            "uuid": key_uuid
        }
    
    def bind_to_asset(self, usb_key_id: int, asset_id: int) -> USBKey:
        """Bind USB key to specific asset"""
        usb_key = self.db.query(USBKey).filter(USBKey.id == usb_key_id).first()
        if not usb_key:
            raise ValueError("USB key not found")
        
        asset = self.db.query(Asset).filter(Asset.id == asset_id).first()
        if not asset:
            raise ValueError("Asset not found")
        
        # Check if already bound
        if usb_key.asset_id:
            raise ValueError(f"USB key already bound to asset {usb_key.asset_id}")
        
        # Check if asset already has a USB key
        existing_key = self.db.query(USBKey).filter(USBKey.asset_id == asset_id).first()
        if existing_key:
            raise ValueError(f"Asset already has USB key {existing_key.uuid}")
        
        usb_key.asset_id = asset_id
        usb_key.status = USBKeyStatus.BOUND
        usb_key.bound_at = datetime.utcnow()
        
        self.db.commit()
        self.db.refresh(usb_key)
        
        return usb_key
    
    def unbind_from_asset(self, usb_key_id: int) -> USBKey:
        """Unbind USB key from asset"""
        usb_key = self.db.query(USBKey).filter(USBKey.id == usb_key_id).first()
        if not usb_key:
            raise ValueError("USB key not found")
        
        usb_key.asset_id = None
        usb_key.status = USBKeyStatus.ACTIVE
        usb_key.bound_at = None
        
        self.db.commit()
        self.db.refresh(usb_key)
        
        return usb_key
    
    def create_backup(self, usb_key_id: int, user_id: Optional[int] = None) -> USBKeyBackup:
        """
        Create encrypted backup of USB key
        可以下载或用于恢复
        """
        usb_key = self.db.query(USBKey).filter(USBKey.id == usb_key_id).first()
        if not usb_key:
            raise ValueError("USB key not found")
        
        # Decrypt key material
        encryption_key = self._derive_key(self.master_key.decode(), bytes.fromhex(usb_key.salt))
        key_material = self._decrypt_data(usb_key.key_material, encryption_key)
        
        # Create backup data
        backup_data = {
            "uuid": usb_key.uuid,
            "label": usb_key.label,
            "key_material": key_material,
            "salt": usb_key.salt,
            "created_at": datetime.utcnow().isoformat()
        }
        
        # Generate backup encryption
        backup_salt = secrets.token_hex(32)
        backup_password = secrets.token_urlsafe(32)
        backup_key = self._derive_key(backup_password, bytes.fromhex(backup_salt))
        encrypted_backup = self._encrypt_data(str(backup_data), backup_key)
        
        # Calculate checksum
        checksum = hashlib.sha256(encrypted_backup.encode()).hexdigest()
        
        # Create backup record
        backup = USBKeyBackup(
            usb_key_id=usb_key_id,
            backup_uuid=str(uuid.uuid4()),
            encrypted_backup=encrypted_backup,
            backup_salt=backup_salt,
            backup_size=len(encrypted_backup),
            checksum=checksum,
            recovery_password=hashlib.sha256(backup_password.encode()).hexdigest(),
            created_by=user_id,
            expires_at=datetime.utcnow() + timedelta(days=365)  # 1 year expiry
        )
        
        self.db.add(backup)
        
        # Update USB key backup info
        usb_key.backup_count += 1
        usb_key.last_backup_at = datetime.utcnow()
        
        self.db.commit()
        self.db.refresh(backup)
        
        return {
            "backup": backup,
            "backup_password": backup_password,  # Save this securely
            "backup_file": base64.b64encode(encrypted_backup.encode()).decode()  # Download this
        }
    
    def restore_from_backup(
        self,
        backup_uuid: str,
        backup_password: str,
        new_label: Optional[str] = None
    ) -> USBKey:
        """
        Restore USB key from backup
        可以在服务节点上执行恢复
        """
        backup = self.db.query(USBKeyBackup).filter(
            USBKeyBackup.backup_uuid == backup_uuid
        ).first()
        
        if not backup:
            raise ValueError("Backup not found")
        
        # Verify backup password
        password_hash = hashlib.sha256(backup_password.encode()).hexdigest()
        if password_hash != backup.recovery_password:
            raise ValueError("Invalid backup password")
        
        # Decrypt backup
        backup_key = self._derive_key(backup_password, bytes.fromhex(backup.backup_salt))
        try:
            backup_data = eval(self._decrypt_data(backup.encrypted_backup, backup_key))
        except Exception as e:
            raise ValueError(f"Failed to decrypt backup: {str(e)}")
        
        # Create new USB key from backup
        label = new_label or f"{backup_data['label']} (Restored)"
        
        usb_key = USBKey(
            uuid=str(uuid.uuid4()),  # New UUID
            label=label,
            key_material=self._encrypt_data(
                backup_data['key_material'],
                self._derive_key(self.master_key.decode(), bytes.fromhex(backup_data['salt']))
            ),
            salt=backup_data['salt'],
            recovery_code=backup.usb_key.recovery_code,  # Copy from original
            status=USBKeyStatus.ACTIVE,
            is_initialized=True,
            description=f"Restored from backup {backup_uuid}"
        )
        
        self.db.add(usb_key)
        
        # Update backup stats
        backup.recovered_count += 1
        backup.last_recovered_at = datetime.utcnow()
        
        self.db.commit()
        self.db.refresh(usb_key)
        
        return usb_key
    
    def rebuild_key(
        self,
        usb_key_id: int,
        recovery_code: str,
        new_label: Optional[str] = None
    ) -> Dict:
        """
        Rebuild USB key using recovery code
        用于USB key丢失或损坏的情况
        """
        usb_key = self.db.query(USBKey).filter(USBKey.id == usb_key_id).first()
        if not usb_key:
            raise ValueError("USB key not found")
        
        # Verify recovery code
        recovery_hash = hashlib.sha256(recovery_code.encode()).hexdigest()
        if recovery_hash != usb_key.recovery_code:
            raise ValueError("Invalid recovery code")
        
        # Generate new key material (保持UUID不变)
        key_material = secrets.token_hex(32)
        new_salt = secrets.token_hex(32)
        
        # Encrypt new key material
        encryption_key = self._derive_key(self.master_key.decode(), bytes.fromhex(new_salt))
        encrypted_key_material = self._encrypt_data(key_material, encryption_key)
        
        # Update USB key
        if new_label:
            usb_key.label = new_label
        usb_key.key_material = encrypted_key_material
        usb_key.salt = new_salt
        usb_key.status = USBKeyStatus.ACTIVE
        usb_key.updated_at = datetime.utcnow()
        
        self.db.commit()
        self.db.refresh(usb_key)
        
        return {
            "usb_key": usb_key,
            "key_material": key_material,  # Write this to new USB key
            "uuid": usb_key.uuid
        }
    
    def revoke_key(self, usb_key_id: int, user_id: Optional[int] = None, reason: Optional[str] = None) -> USBKey:
        """Revoke USB key (security breach, loss, etc.)"""
        usb_key = self.db.query(USBKey).filter(USBKey.id == usb_key_id).first()
        if not usb_key:
            raise ValueError("USB key not found")
        
        usb_key.status = USBKeyStatus.REVOKED
        usb_key.revoked_at = datetime.utcnow()
        usb_key.revoked_by = user_id
        if reason:
            usb_key.description = f"{usb_key.description or ''}\nRevoked: {reason}"
        
        self.db.commit()
        self.db.refresh(usb_key)
        
        return usb_key
    
    def record_usage(self, usb_key_id: int, success: bool = True):
        """Record USB key usage"""
        usb_key = self.db.query(USBKey).filter(USBKey.id == usb_key_id).first()
        if not usb_key:
            return
        
        usb_key.last_used_at = datetime.utcnow()
        if success:
            usb_key.use_count += 1
            usb_key.failed_attempts = 0
        else:
            usb_key.failed_attempts += 1
        
        self.db.commit()
    
    def get_key_for_asset(self, asset_id: int) -> Optional[USBKey]:
        """Get USB key bound to asset"""
        return self.db.query(USBKey).filter(
            USBKey.asset_id == asset_id,
            USBKey.status.in_([USBKeyStatus.ACTIVE, USBKeyStatus.BOUND])
        ).first()
    
    def list_keys(
        self,
        status: Optional[USBKeyStatus] = None,
        asset_id: Optional[int] = None,
        skip: int = 0,
        limit: int = 100
    ) -> List[USBKey]:
        """List USB keys with filtering"""
        query = self.db.query(USBKey)
        
        if status:
            query = query.filter(USBKey.status == status)
        
        if asset_id:
            query = query.filter(USBKey.asset_id == asset_id)
        
        return query.offset(skip).limit(limit).all()
