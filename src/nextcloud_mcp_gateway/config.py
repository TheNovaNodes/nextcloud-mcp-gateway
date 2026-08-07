"""Configuration provider for Nextcloud MCP Gateway."""

import os
from dataclasses import dataclass

@dataclass
class NextcloudConfig:
    nc_url: str = os.getenv("NC_URL", "http://127.0.0.1:8080").rstrip("/")
    public_url: str = os.getenv("NC_PUBLIC_URL", "https://nc.shtab-ai.ru").rstrip("/")
    username: str = os.getenv("NC_USER", "")
    password: str = os.getenv("NC_APP_PASSWORD", "")
    timeout: float = float(os.getenv("NC_TIMEOUT", "30.0"))

    @property
    def webdav_url(self) -> str:
        """Construct WebDAV base endpoint for user."""
        user = self.username or "remote.php/webdav"
        if self.username:
            return f"{self.nc_url}/remote.php/dav/files/{user}"
        return f"{self.nc_url}/remote.php/webdav"

    @property
    def ocs_url(self) -> str:
        """Construct OCS API base endpoint."""
        return f"{self.nc_url}/ocs/v1.php/cloud"

def get_config() -> NextcloudConfig:
    """Retrieve runtime Nextcloud configuration."""
    return NextcloudConfig()
