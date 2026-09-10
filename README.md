```yaml
module_type: mcp_server
status: active
protocol: mcp
primary_capability: nextcloud_webdav_caldav_deck
requires: nextcloud
works_with: claude_desktop, ai_agents, mcp_router
last_verified: 2026-09-10
```

# nextcloud-mcp-gateway

**Nextcloud Data Plane MCP Server for AI Agents to read, write, and manage files, CalDAV calendars, and Deck boards via WebDAV and OCS API (Go 1.22+).**

## Status and Last Verified Date
- **Status:** Active (v2.0.0 Go rewrite)
- **Last Verified Date:** 2026-09-10

## What it does / does not do
- **What it does:**  
  Exposes Nextcloud file operations (WebDAV), user information (OCS API), Kanban tasks (Nextcloud Deck), and calendar events (CalDAV) to AI agents via the Model Context Protocol (MCP). It allows listing, reading, creating, and deleting files and folders, checking instance health and user quotas, and organizing Deck cards and CalDAV events.
- **What it does not do:**  
  It does not administer the Nextcloud instance (e.g., creating users, configuring server settings), nor does it provide a full web GUI. It handles data-plane operations scoped to the authenticated user.

## Why an agent would use it
An AI agent can use this gateway to interact with Nextcloud storage, read documents, write reports, organize files into folders, manage Deck Kanban tasks, and schedule events via CalDAV.

## Architecture and Dependencies
- **Language:** Go 1.22+
- **Framework:** `github.com/mark3labs/mcp-go`
- **APIs Used:** Nextcloud WebDAV (`/remote.php/dav/files/{user}`), CalDAV (`/remote.php/dav/calendars/{user}/{calendar}`), Deck REST API v1.0, and OCS REST API (`/ocs/v1.php/cloud`).
- **Execution:** Runs as a standard stdio MCP server compiled as a single static binary.

## Quick Start and Health Check
```bash
# Clone the repository
git clone https://github.com/TheNovaNodes/nextcloud-mcp-gateway.git
cd nextcloud-mcp-gateway

# Build binary
make build

# Run unit tests
make test

# Start server
./bin/nextcloud-mcp-gateway
```
**Health Check:** Agents can call the `nextcloud_health` tool which checks the `/status.php` endpoint of the configured Nextcloud instance.

## Configuration and Environment Variables
Copy `.env.example` to `.env` and fill in credentials.

- `NC_URL`: Local or private Nextcloud instance endpoint (default: `http://127.0.0.1:8080`)
- `NC_PUBLIC_URL`: Public HTTPS domain endpoint (default: `https://nextcloud.example.com`)
- `NC_USER`: Nextcloud username
- `NC_APP_PASSWORD`: Nextcloud App Password or WebDAV token
- `NC_TIMEOUT`: Request timeout in seconds (default: `30.0`)

## MCP Tools Reference

### Files & Storage (WebDAV)
- `list_files(path, offset, limit)`: Lists files and folders in a Nextcloud directory via WebDAV `PROPFIND` (Depth: 1) with pagination.
- `read_file(path)`: Reads textual content of a file from Nextcloud storage (enforces 10MB limit to prevent OOM).
- `write_file(path, content)`: **[HITL protected]** Stages file creation or overwrite. Requires user approval.
- `delete_file(path)`: **[HITL protected]** Stages deletion of a file or folder. Requires user approval.
- `create_folder(path)`: **[HITL protected]** Stages folder creation via WebDAV `MKCOL`. Requires user approval.

### Security & Approvals (HITL)
- `execute_pending_action(token)`: Approves and executes an action guarded by HITL (valid for 5 minutes).

### Calendar (CalDAV)
- `list_calendar_events(calendar_name)`: Lists events from a Nextcloud CalDAV calendar via `REPORT`.
- `create_calendar_event(event_uid, summary, dtstart, dtend, calendar_name)`: Creates a new event using iCalendar (.ics).
- `delete_calendar_event(event_uid, calendar_name)`: Deletes an event by UID.

### Kanban (Nextcloud Deck)
- `list_deck_boards()`: Lists all Kanban boards available to the user.
- `list_deck_stacks(board_id)`: Lists all columns (stacks) on a Deck board.
- `create_deck_card(board_id, stack_id, title, description)`: Creates a new card in a stack.
- `update_deck_card(board_id, stack_id, card_id, title, description, order)`: Updates or moves a Kanban card.
- `delete_deck_card(board_id, stack_id, card_id)`: Deletes a card.

### System & Diagnostics
- `nextcloud_health()`: Checks Nextcloud instance health and status.
- `get_user_info()`: Retrieves user quota and profile via OCS API.

## Tests
Run tests with race detection and coverage:
```bash
make test
make coverage
```

## License
MIT License. See [LICENSE](LICENSE) for details.
