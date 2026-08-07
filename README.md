# nextcloud-mcp-gateway ☁️

[![Python 3.10+](https://img.shields.io/badge/python-3.10%2B-blue)](https://www.python.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![MCP Server](https://img.shields.io/badge/MCP--Server-available-green)](https://modelcontextprotocol.io/)
[![Nextcloud](https://img.shields.io/badge/Nextcloud-30-0082c9.svg)](https://nextcloud.com/)
[![Status: Active](https://img.shields.io/badge/Status-Active-brightgreen)]

## About

High-performance **Model Context Protocol (MCP)** Data Plane server for Nextcloud integration (`TheNovaNodes/nextcloud-mcp-gateway`). Enables AI agents to read, write, organize, and manage user files, documentation, reports, and CRM assets over WebDAV and Nextcloud OCS REST API.

---

## Exposed MCP Tools

| Tool | Parameters | Description |
| :--- | :--- | :--- |
| **`list_files`** | `path: str = "/"` | Lists files and directories with timestamps, sizes, and MIME types via WebDAV `PROPFIND`. |
| **`read_file`** | `path: str` | Reads textual or markdown document content directly from Nextcloud storage. |
| **`write_file`** | `path: str, content: str` | Creates or updates user files in Nextcloud storage via WebDAV `PUT`. |
| **`delete_file`** | `path: str` | Deletes files or folders from storage via WebDAV `DELETE`. |
| **`create_folder`** | `path: str` | Creates new directory structures in Nextcloud storage via WebDAV `MKCOL`. |
| **`get_user_info`** | *None* | Retrieves user storage quota, display name, and details via OCS API. |
| **`nextcloud_health`**| *None* | Runs diagnostic health checks on `/status.php` and WebDAV availability. |

---

## Installation & Setup

```bash
# Clone repository
git clone https://github.com/TheNovaNodes/nextcloud-mcp-gateway.git
cd nextcloud-mcp-gateway

# Install in editable mode
pip install -e .
```

---

## Configuration

Copy `.env.example` to `.env` and fill in credentials:

```bash
cp .env.example .env
```

| Variable | Description | Default |
| :--- | :--- | :--- |
| `NC_URL` | Local Nextcloud instance endpoint | `http://127.0.0.1:8080` |
| `NC_PUBLIC_URL` | Public HTTPS domain endpoint | `https://nc.shtab-ai.ru` |
| `NC_USER` | Nextcloud username | `zavlab` |
| `NC_APP_PASSWORD`| Nextcloud App Password or WebDAV token | *Required* |
| `NC_TIMEOUT` | Request timeout in seconds | `30` |

---

## MCP Client Integration

Add to your MCP configuration (`mcp_config.json` or `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "nextcloud-gateway": {
      "command": "python3",
      "args": ["-m", "nextcloud_mcp_gateway.server"],
      "cwd": "/home/ddoctorm/projects/TheNovaNodes/nextcloud-mcp-gateway",
      "env": {
        "NC_URL": "http://127.0.0.1:8080",
        "NC_PUBLIC_URL": "https://nc.shtab-ai.ru",
        "NC_USER": "zavlab",
        "NC_APP_PASSWORD": "your_app_password"
      }
    }
  }
}
```

---

## Running Unit Tests

```bash
python3 -m pytest -v
```

---

## License

MIT — See [LICENSE](LICENSE) file.
