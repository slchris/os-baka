"""
PXE Docker Service Manager
Manages Docker container for PXE/TFTP/DHCP services
"""
import os
import subprocess
import logging
from typing import Dict, Optional
from datetime import datetime
from sqlalchemy.orm import Session
from app.models.pxe_service_config import PXEServiceConfig

logger = logging.getLogger(__name__)


class PXEDockerService:
    """Manages PXE service Docker container"""
    
    def __init__(self, db: Session):
        self.db = db
        self.config = self._get_or_create_config()
    
    def _get_or_create_config(self) -> PXEServiceConfig:
        """Get singleton config or create default"""
        config = self.db.query(PXEServiceConfig).first()
        if not config:
            config = PXEServiceConfig(
                server_ip="192.168.1.1",
                dhcp_range_start="192.168.1.100",
                dhcp_range_end="192.168.1.200",
                service_enabled=False,
                container_status="stopped"
            )
            self.db.add(config)
            self.db.commit()
            self.db.refresh(config)
        return config
    
    def get_config(self) -> PXEServiceConfig:
        """Get current configuration"""
        return self.config
    
    def update_config(self, **kwargs) -> PXEServiceConfig:
        """Update configuration"""
        for key, value in kwargs.items():
            if hasattr(self.config, key):
                setattr(self.config, key, value)
        
        self.config.updated_at = datetime.utcnow()
        self.db.commit()
        self.db.refresh(self.config)
        return self.config
    
    def _run_docker_command(self, args: list) -> Dict[str, any]:
        """Execute docker command"""
        try:
            result = subprocess.run(
                ["docker"] + args,
                capture_output=True,
                text=True,
                timeout=30
            )
            return {
                "success": result.returncode == 0,
                "stdout": result.stdout.strip(),
                "stderr": result.stderr.strip(),
                "returncode": result.returncode
            }
        except subprocess.TimeoutExpired:
            return {
                "success": False,
                "stdout": "",
                "stderr": "Command timed out",
                "returncode": -1
            }
        except Exception as e:
            return {
                "success": False,
                "stdout": "",
                "stderr": str(e),
                "returncode": -1
            }
    
    def _container_exists(self) -> bool:
        """Check if container exists"""
        result = self._run_docker_command([
            "ps", "-a", "--filter", f"name={self.config.container_name}", "--format", "{{.Names}}"
        ])
        return result["success"] and self.config.container_name in result["stdout"]
    
    def _container_running(self) -> bool:
        """Check if container is running"""
        result = self._run_docker_command([
            "ps", "--filter", f"name={self.config.container_name}", "--format", "{{.Names}}"
        ])
        return result["success"] and self.config.container_name in result["stdout"]
    
    def _generate_dnsmasq_conf(self) -> str:
        """Generate dnsmasq configuration"""
        conf_lines = [
            "# PXE Server Configuration",
            f"interface=eth0",
            f"bind-interfaces",
            "",
            "# DHCP Configuration",
            f"dhcp-range={self.config.dhcp_range_start},{self.config.dhcp_range_end},12h",
            f"dhcp-option=option:netmask,{self.config.netmask}",
        ]
        
        if self.config.gateway:
            conf_lines.append(f"dhcp-option=option:router,{self.config.gateway}")
        
        if self.config.dns_servers:
            conf_lines.append(f"dhcp-option=option:dns-server,{self.config.dns_servers}")
        
        conf_lines.extend([
            "",
            "# TFTP Configuration",
            "enable-tftp",
            f"tftp-root={self.config.tftp_root}",
            ""
        ])
        
        # BIOS Boot
        if self.config.enable_bios:
            conf_lines.extend([
                "# BIOS PXE Boot",
                f"dhcp-boot={self.config.bios_boot_file}",
                ""
            ])
        
        # UEFI Boot  
        if self.config.enable_uefi:
            conf_lines.extend([
                "# UEFI PXE Boot",
                f"dhcp-match=set:efi-x86_64,option:client-arch,7",
                f"dhcp-match=set:efi-x86_64,option:client-arch,9",
                f"dhcp-boot=tag:efi-x86_64,{self.config.uefi_boot_file}",
                ""
            ])
        
        conf_lines.extend([
            "# Logging",
            "log-dhcp",
            "log-queries",
        ])
        
        return "\n".join(conf_lines)
    
    def start_service(self) -> Dict[str, any]:
        """Start PXE Docker container"""
        try:
            # Create TFTP root if not exists
            os.makedirs(self.config.tftp_root, exist_ok=True)
            
            # Generate dnsmasq.conf
            dnsmasq_conf = self._generate_dnsmasq_conf()
            conf_path = os.path.join(self.config.tftp_root, "dnsmasq.conf")
            with open(conf_path, 'w') as f:
                f.write(dnsmasq_conf)
            
            # Check if container exists
            if self._container_exists():
                # Container exists, just start it
                if self._container_running():
                    return {
                        "success": True,
                        "message": "Container already running",
                        "status": "running"
                    }
                
                result = self._run_docker_command(["start", self.config.container_name])
                if result["success"]:
                    self.update_config(
                        container_status="running",
                        service_enabled=True,
                        last_started=datetime.utcnow()
                    )
                    return {
                        "success": True,
                        "message": "Container started successfully",
                        "status": "running"
                    }
                else:
                    return {
                        "success": False,
                        "message": f"Failed to start container: {result['stderr']}",
                        "status": "error"
                    }
            
            # Create new container
            docker_args = [
                "run", "-d",
                "--name", self.config.container_name,
                "--network", "host",
                "--cap-add", "NET_ADMIN",
                "-v", f"{self.config.tftp_root}:/srv/tftp",
                "-v", f"{conf_path}:/etc/dnsmasq.conf:ro",
                self.config.container_image,
                "dnsmasq", "--no-daemon", "-C", "/etc/dnsmasq.conf"
            ]
            
            result = self._run_docker_command(docker_args)
            
            if result["success"]:
                container_id = result["stdout"]
                self.update_config(
                    container_id=container_id,
                    container_status="running",
                    service_enabled=True,
                    last_started=datetime.utcnow()
                )
                return {
                    "success": True,
                    "message": "Container created and started",
                    "container_id": container_id,
                    "status": "running"
                }
            else:
                self.update_config(container_status="error")
                return {
                    "success": False,
                    "message": f"Failed to create container: {result['stderr']}",
                    "status": "error"
                }
        
        except Exception as e:
            logger.error(f"Failed to start PXE service: {e}")
            self.update_config(container_status="error")
            return {
                "success": False,
                "message": str(e),
                "status": "error"
            }
    
    def stop_service(self) -> Dict[str, any]:
        """Stop PXE Docker container"""
        try:
            if not self._container_exists():
                return {
                    "success": True,
                    "message": "Container does not exist",
                    "status": "stopped"
                }
            
            if not self._container_running():
                self.update_config(
                    container_status="stopped",
                    service_enabled=False,
                    last_stopped=datetime.utcnow()
                )
                return {
                    "success": True,
                    "message": "Container already stopped",
                    "status": "stopped"
                }
            
            result = self._run_docker_command(["stop", self.config.container_name])
            
            if result["success"]:
                self.update_config(
                    container_status="stopped",
                    service_enabled=False,
                    last_stopped=datetime.utcnow()
                )
                return {
                    "success": True,
                    "message": "Container stopped successfully",
                    "status": "stopped"
                }
            else:
                return {
                    "success": False,
                    "message": f"Failed to stop container: {result['stderr']}",
                    "status": "error"
                }
        
        except Exception as e:
            logger.error(f"Failed to stop PXE service: {e}")
            return {
                "success": False,
                "message": str(e),
                "status": "error"
            }
    
    def restart_service(self) -> Dict[str, any]:
        """Restart PXE Docker container"""
        stop_result = self.stop_service()
        if not stop_result["success"]:
            return stop_result
        
        return self.start_service()
    
    def get_status(self) -> Dict[str, any]:
        """Get service status"""
        container_running = self._container_running()
        container_exists = self._container_exists()
        
        if container_running:
            status = "running"
        elif container_exists:
            status = "stopped"
        else:
            status = "not_created"
        
        # Update config if status changed
        if self.config.container_status != status:
            self.update_config(container_status=status)
        
        return {
            "status": status,
            "container_exists": container_exists,
            "container_running": container_running,
            "enabled": self.config.service_enabled,
            "last_started": self.config.last_started.isoformat() if self.config.last_started else None,
            "last_stopped": self.config.last_stopped.isoformat() if self.config.last_stopped else None
        }
    
    def get_logs(self, tail: int = 100) -> Dict[str, any]:
        """Get container logs"""
        if not self._container_exists():
            return {
                "success": False,
                "message": "Container does not exist",
                "logs": ""
            }
        
        result = self._run_docker_command([
            "logs", "--tail", str(tail), self.config.container_name
        ])
        
        return {
            "success": result["success"],
            "logs": result["stdout"] + "\n" + result["stderr"]
        }
    
    def remove_container(self) -> Dict[str, any]:
        """Remove container (must be stopped first)"""
        try:
            if self._container_running():
                return {
                    "success": False,
                    "message": "Container is running. Stop it first.",
                    "status": "error"
                }
            
            if not self._container_exists():
                return {
                    "success": True,
                    "message": "Container does not exist",
                    "status": "removed"
                }
            
            result = self._run_docker_command(["rm", self.config.container_name])
            
            if result["success"]:
                self.update_config(
                    container_id=None,
                    container_status="stopped"
                )
                return {
                    "success": True,
                    "message": "Container removed successfully",
                    "status": "removed"
                }
            else:
                return {
                    "success": False,
                    "message": f"Failed to remove container: {result['stderr']}",
                    "status": "error"
                }
        
        except Exception as e:
            logger.error(f"Failed to remove container: {e}")
            return {
                "success": False,
                "message": str(e),
                "status": "error"
            }
