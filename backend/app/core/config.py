from pydantic_settings import BaseSettings
from typing import List, Union


class Settings(BaseSettings):
    """Application settings"""
    
    # Database
    DATABASE_URL: str = "postgresql://osbaka:password@localhost:5432/osbaka"
    
    # Redis
    REDIS_URL: str = "redis://localhost:6379/0"
    
    # Security
    SECRET_KEY: str = "your-secret-key-here-change-in-production"
    ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30
    
    # CORS - can be string (comma-separated) or list
    CORS_ORIGINS: Union[str, List[str]] = "http://localhost:5173,http://localhost:3000"
    
    @property
    def cors_origins_list(self) -> List[str]:
        """Convert CORS_ORIGINS to list"""
        if isinstance(self.CORS_ORIGINS, str):
            return [origin.strip() for origin in self.CORS_ORIGINS.split(',')]
        return self.CORS_ORIGINS
    
    # PXE Service
    PXE_CONFIG_DIR: str = "/etc/pxe"
    TFTP_ROOT: str = "/var/lib/tftpboot"
    DHCP_RANGE_START: str = "192.168.1.100"
    DHCP_RANGE_END: str = "192.168.1.200"
    
    class Config:
        env_file = ".env"
        case_sensitive = True


settings = Settings()
