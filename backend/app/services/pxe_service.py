"""
PXE Configuration Service
Handles PXE configuration, dnsmasq config generation, and service management
"""
import os
import subprocess
from typing import Optional
from pathlib import Path
from sqlalchemy.orm import Session
from app.models.pxe_config import PXEConfig, PXEDeployment
from app.schemas.pxe_config import PXEConfigCreate, PXEConfigUpdate
from app.core.config import settings


class PXEService:
    """Service for managing PXE configurations"""
    
    def __init__(self, db: Session):
        self.db = db
        self.dnsmasq_conf_dir = Path(settings.PXE_CONFIG_DIR) / "dnsmasq.d"
        self.dnsmasq_conf_dir.mkdir(parents=True, exist_ok=True)
        
    def create_config(self, config_data: PXEConfigCreate) -> PXEConfig:
        """Create new PXE configuration"""
        config = PXEConfig(**config_data.model_dump())
        self.db.add(config)
        self.db.commit()
        self.db.refresh(config)
        
        # Generate dnsmasq configuration
        if config.enabled:
            self.generate_dnsmasq_config(config)
        
        return config
    
    def update_config(self, config_id: int, config_data: PXEConfigUpdate) -> Optional[PXEConfig]:
        """Update PXE configuration"""
        config = self.db.query(PXEConfig).filter(PXEConfig.id == config_id).first()
        if not config:
            return None
        
        # Update fields
        for field, value in config_data.model_dump(exclude_unset=True).items():
            setattr(config, field, value)
        
        self.db.commit()
        self.db.refresh(config)
        
        # Regenerate dnsmasq configuration
        self.generate_dnsmasq_config(config)
        
        return config
    
    def delete_config(self, config_id: int) -> bool:
        """Delete PXE configuration"""
        config = self.db.query(PXEConfig).filter(PXEConfig.id == config_id).first()
        if not config:
            return False
        
        # Remove dnsmasq configuration file
        self.remove_dnsmasq_config(config)
        
        self.db.delete(config)
        self.db.commit()
        return True
    
    def generate_dnsmasq_config(self, config: PXEConfig) -> str:
        """
        Generate dnsmasq configuration for a PXE config
        Creates a file like: /etc/dnsmasq.d/pxe-<hostname>.conf
        """
        if not config.enabled:
            # Remove config if disabled
            self.remove_dnsmasq_config(config)
            return ""
        
        conf_file = self.dnsmasq_conf_dir / f"pxe-{config.hostname}.conf"
        
        # Dnsmasq configuration template
        conf_content = f"""# PXE Configuration for {config.hostname}
# Auto-generated - do not edit manually
# MAC: {config.mac_address}

# DHCP host configuration
dhcp-host={config.mac_address},{config.ip_address},{config.hostname}

"""
        
        # Add boot configuration if present
        if config.boot_image:
            conf_content += f"""# PXE Boot configuration
dhcp-boot=tag:{config.mac_address.replace(':', '')},{config.boot_image}

"""
        
        # Add additional boot params
        if config.boot_params:
            conf_content += f"""# Boot parameters
{config.boot_params}

"""
        
        # Write configuration file
        conf_file.write_text(conf_content)
        return conf_content
    
    def remove_dnsmasq_config(self, config: PXEConfig) -> None:
        """Remove dnsmasq configuration file"""
        conf_file = self.dnsmasq_conf_dir / f"pxe-{config.hostname}.conf"
        if conf_file.exists():
            conf_file.unlink()
    
    def generate_all_configs(self) -> list[str]:
        """Generate dnsmasq configs for all enabled PXE configurations"""
        configs = self.db.query(PXEConfig).filter(PXEConfig.enabled == True).all()
        generated = []
        
        for config in configs:
            content = self.generate_dnsmasq_config(config)
            generated.append(f"{config.hostname}: {len(content)} bytes")
        
        return generated
    
    def validate_dnsmasq_config(self) -> dict:
        """Validate dnsmasq configuration"""
        try:
            # Run dnsmasq --test to validate configuration
            result = subprocess.run(
                ['dnsmasq', '--test'],
                capture_output=True,
                text=True,
                timeout=5
            )
            
            return {
                'valid': result.returncode == 0,
                'output': result.stdout,
                'errors': result.stderr.split('\n') if result.stderr else []
            }
        except FileNotFoundError:
            return {
                'valid': False,
                'output': '',
                'errors': ['dnsmasq command not found - is it installed?']
            }
        except subprocess.TimeoutExpired:
            return {
                'valid': False,
                'output': '',
                'errors': ['dnsmasq validation timed out']
            }
        except Exception as e:
            return {
                'valid': False,
                'output': '',
                'errors': [str(e)]
            }
    
    def restart_dnsmasq(self) -> dict:
        """Restart dnsmasq service"""
        try:
            # Try systemd first
            result = subprocess.run(
                ['systemctl', 'restart', 'dnsmasq'],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            if result.returncode == 0:
                return {
                    'success': True,
                    'service': 'dnsmasq',
                    'action': 'restart',
                    'message': 'dnsmasq service restarted successfully',
                    'output': result.stdout
                }
            else:
                return {
                    'success': False,
                    'service': 'dnsmasq',
                    'action': 'restart',
                    'message': f'Failed to restart dnsmasq: {result.stderr}',
                    'output': result.stdout
                }
        except FileNotFoundError:
            # Try alternative service command
            try:
                result = subprocess.run(
                    ['service', 'dnsmasq', 'restart'],
                    capture_output=True,
                    text=True,
                    timeout=30
                )
                return {
                    'success': result.returncode == 0,
                    'service': 'dnsmasq',
                    'action': 'restart',
                    'message': 'dnsmasq restarted' if result.returncode == 0 else result.stderr,
                    'output': result.stdout
                }
            except Exception as e:
                return {
                    'success': False,
                    'service': 'dnsmasq',
                    'action': 'restart',
                    'message': f'Failed to restart service: {str(e)}',
                    'output': None
                }
        except subprocess.TimeoutExpired:
            return {
                'success': False,
                'service': 'dnsmasq',
                'action': 'restart',
                'message': 'Service restart timed out',
                'output': None
            }
        except Exception as e:
            return {
                'success': False,
                'service': 'dnsmasq',
                'action': 'restart',
                'message': str(e),
                'output': None
            }
    
    def get_dnsmasq_status(self) -> dict:
        """Get dnsmasq service status"""
        try:
            result = subprocess.run(
                ['systemctl', 'status', 'dnsmasq'],
                capture_output=True,
                text=True,
                timeout=5
            )
            
            # Parse status output
            is_active = 'active (running)' in result.stdout
            
            return {
                'service': 'dnsmasq',
                'running': is_active,
                'status': 'active' if is_active else 'inactive',
                'output': result.stdout
            }
        except Exception as e:
            return {
                'service': 'dnsmasq',
                'running': False,
                'status': 'unknown',
                'output': str(e)
            }
    
    def create_deployment(self, pxe_config_id: int, user_id: Optional[int] = None) -> PXEDeployment:
        """Create a new deployment record"""
        deployment = PXEDeployment(
            pxe_config_id=pxe_config_id,
            status='pending',
            initiated_by=user_id
        )
        self.db.add(deployment)
        self.db.commit()
        self.db.refresh(deployment)
        return deployment
