```yaml
module_type: mcp_server
status: active
protocol: mcp
primary_capability: nextcloud_webdav_ocs
requires: nextcloud, httpx
works_with: claude_desktop, ai_agents
last_verified: 2026-08-21
```

# nextcloud-mcp-gateway

**Nextcloud Data Plane MCP Server for AI Agents to read, write, and manage files via WebDAV and OCS API.**

## Status and Last Verified Date
**Status:** Active  
**Last Verified Date:** 2026-08-21

## What it does / does not do
**What it does:**  
Exposes Nextcloud file operations (WebDAV) and user information (OCS API) to AI agents via the Model Context Protocol (MCP). It allows listing, reading, creating, and deleting files and folders, checking instance health and user quotas, and interacting with Nextcloud Deck (Kanban boards and cards) and Calendars (CalDAV).

**What it does not do:**  
It does not administer the Nextcloud instance (e.g., creating users, configuring plugins, changing server settings), nor does it provide a full web GUI. It handles only data-plane operations scoped to the authenticated user.

## Why an agent would use it
An AI agent can use this gateway to interact with a user's Nextcloud storage. Agents can read documents or code stored in Nextcloud, generate reports, write logs, organize files into folders, and delete temporary resources. With Deck and Calendar integration, agents can act as personal assistants, organizing tasks and scheduling events, seamlessly integrating AI workflows with the user's personal cloud.

## Architecture and Dependencies
- **Language:** Python 3.10+
- **Frameworks/Libraries:** `mcp` (FastMCP framework), `httpx` (async HTTP client), `pydantic`.
- **APIs Used:** Nextcloud WebDAV (`/remote.php/webdav`) and Nextcloud OCS REST API (`/ocs/v1.php/cloud`).
- **Execution:** Runs as a standard stdio MCP server.

## Compatibility
Compatible with Python 3.10+ and any standard MCP client (e.g., Claude Desktop). Requires a running Nextcloud instance reachable via HTTP/HTTPS.

## Quick Start and Health Check
```bash
# Clone the repository
git clone https://github.com/TheNovaNodes/nextcloud-mcp-gateway.git
cd nextcloud-mcp-gateway

# Install dependencies and the CLI tool
pip install -e .

# The server can be started using the CLI entry point
nextcloud-mcp-gateway
```
**Health Check:** Agents can call the `nextcloud_health` tool which checks the `/status.php` endpoint of the configured Nextcloud instance.

## Configuration and Environment Variables
Copy `.env.example` to `.env` and fill in credentials.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `NC_URL` | Local or private Nextcloud instance endpoint | `http://127.0.0.1:8080` |
| `NC_PUBLIC_URL` | Public HTTPS domain endpoint | `https://nc.shtab-ai.ru` |
| `NC_USER` | Nextcloud username | `""` |
| `NC_APP_PASSWORD`| Nextcloud App Password or WebDAV token | `""` |
| `NC_TIMEOUT` | Request timeout in seconds | `30.0` |

## Complete MCP Tool/API Table with Side Effects
| Tool Name | Parameters | Side Effects | Description |
| :--- | :--- | :--- | :--- |
| `nextcloud_health` | None | **None (Read-only)** | Checks Nextcloud instance health, version, and `status.php` availability. |
| `list_files` | `path: str = "/"`, `offset: int = 0`, `limit: int = 50` | **None (Read-only)** | Lists files and folders in a Nextcloud directory via WebDAV `PROPFIND` (Depth: 1). Paginated. |
| `read_file` | `path: str` | **None (Read-only)** | Reads the textual content of a file from Nextcloud storage. |
| `write_file` | `path: str, content: str` | **Writes Data (HITL)** | Creates or overwrites a file in Nextcloud storage via WebDAV `PUT` (UTF-8). Requires HITL approval. |
| `delete_file` | `path: str` | **Deletes Data (HITL)** | Deletes a file or folder from Nextcloud storage via WebDAV `DELETE`. Requires HITL approval. |
| `create_folder` | `path: str` | **Creates Data (HITL)** | Creates a new folder in Nextcloud storage via WebDAV `MKCOL`. Requires HITL approval. |
| `get_user_info` | None | **None (Read-only)** | Retrieves user storage quota, display name, and details via Nextcloud OCS API. |
| `list_deck_boards` | None | **None (Read-only)** | Lists all Nextcloud Deck Kanban boards available to the user. |
| `create_deck_card` | `board_id: int`, `stack_id: int`, `title: str`, `description: str` | **Creates Data** | Creates a new Kanban card in a Nextcloud Deck. |
| `list_calendar_events` | `calendar_name: str = "personal"` | **None (Read-only)** | Lists events from a Nextcloud CalDAV calendar. |
| `create_calendar_event` | `event_uid: str`, `summary: str`, `dtstart: str`, `dtend: str`, `calendar_name: str` | **Creates Data** | Creates a new event in Nextcloud CalDAV calendar using iCalendar (.ics). |
| `execute_pending_action`| `token: str` | **Executes Data** | Approves and executes an action guarded by HITL (e.g. file deletion/creation). |

## Security Model and Trust Boundaries
- **Authentication:** Relies on HTTP Basic Authentication using `NC_USER` and `NC_APP_PASSWORD`. It is highly recommended to use Nextcloud App Passwords rather than main account passwords.
- **Authorization:** Operations are limited entirely by the permissions of the authenticated user in Nextcloud.
- **Network Boundaries:** The server must be able to reach `NC_URL` over the network. Credentials are sent in HTTP headers (use HTTPS for `NC_URL` in production).

## Tests and Exact Commands
Unit tests are written using `pytest` and `pytest-asyncio`. Run them with:
```bash
python3 -m pytest -v
```

## Operations, Logs, Backup/Restore, Rollback
- **Operations:** The gateway itself is a stateless service. 
- **Logs:** Handled via the MCP client's standard error stream.
- **Backup/Restore:** Since the server is stateless, no local backup is necessary. Actual file data should be backed up within the Nextcloud ecosystem.
- **Rollback:** Roll back to a previous git commit or package version if an update fails.

## Generic MCP-Client Example
```json
{
  "mcpServers": {
    "nextcloud-gateway": {
      "command": "nextcloud-mcp-gateway",
      "env": {
        "NC_URL": "https://your-nextcloud.com",
        "NC_USER": "agent_user",
        "NC_APP_PASSWORD": "your_app_password"
      }
    }
  }
}
```

## Limitations and Roadmap
- **Limitations:** 
  - `read_file` only supports textual data and decodes via standard text formats. Binary files may fail or corrupt if requested this way.
  - `list_files` is limited to a depth of 1 (current directory only).
- **Roadmap:**
  - Add support for binary file transfers.
  - Support recursive folder listings.
  - Enhanced error handling for large file transfers.

## Related TheNovaNodes Modules
Part of the TheNovaNodes AI infrastructure integrations, providing seamless data plane access.

## License
MIT License
