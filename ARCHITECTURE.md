# ☁️ Architecture Documentation: Nextcloud MCP Gateway

`nextcloud-mcp-gateway` provides an asynchronous Data Plane Model Context Protocol (MCP) bridge into Nextcloud, enabling AI agents to read, write, organize, and inspect user storage, Kanban tasks (Deck), and calendar events (CalDAV).

---

## 🏗 High-Level Architecture

```mermaid
graph TD
    Agent[🤖 AI Agent / Model] -->|MCP Protocol| FastMCP[⚡ Nextcloud MCP Gateway]
    FastMCP -->|HITL Guard / UUID Token Verification| Guard{🛡️ Destructive Action?}
    Guard -->|No: Read-Only| API_Router[API Router]
    Guard -->|Yes: Requires execute_pending_action| Pending[Pending Actions Store]
    Pending -->|Token Approved| API_Router
    
    API_Router -->|Async WebDAV / PROPFIND / PUT| WebDAV[📂 Nextcloud WebDAV API]
    API_Router -->|CalDAV / REPORT / PUT / DELETE| CalDAV[📅 Nextcloud CalDAV API]
    API_Router -->|REST / JSON| Deck[🗃️ Nextcloud Deck API]
    API_Router -->|REST / JSON| OCS[👥 Nextcloud OCS Cloud API]
    
    WebDAV -->|Port 8080 / HTTPS nc.shtab-ai.ru| Nextcloud[☁️ Nextcloud Application Server]
    CalDAV --> Nextcloud
    Deck --> Nextcloud
    OCS --> Nextcloud
    
    Nextcloud --> Storage[(Filesystem & Object Storage)]
    Nextcloud --> DB[(MariaDB Database)]
    Nextcloud --> Cache[(Redis Cache & Session Store)]
```

---

## 🔄 Sequence Workflows

### 1. Read Operations (Direct Execution)

```mermaid
sequenceDiagram
    autonumber
    actor Agent as 🤖 AI Agent
    participant MCP as ⚡ nextcloud-mcp-gateway
    participant NC as ☁️ Nextcloud Instance
    participant Store as 💾 Storage Backend

    Agent->>MCP: read_file(path="/Documents/spec.md")
    MCP->>NC: GET /remote.php/dav/files/user/Documents/spec.md
    NC->>Store: Fetch file bytes
    Store-->>NC: Stream binary/text
    NC-->>MCP: HTTP 200 (Text Content)
    MCP-->>Agent: {"status": "success", "content": "..."}
```

### 2. Destructive Operations (Human-In-The-Loop / HITL Guard)

```mermaid
sequenceDiagram
    autonumber
    actor Agent as 🤖 AI Agent
    actor User as 👤 ZaVLab / Operator
    participant MCP as ⚡ nextcloud-mcp-gateway
    participant NC as ☁️ Nextcloud Instance

    Agent->>MCP: write_file(path="/Reports/summary.md", content="...")
    MCP-->>Agent: {"status": "pending_approval", "token": "uuid-v4-token"}
    Agent->>User: Request approval for writing file
    User-->>Agent: Approved
    Agent->>MCP: execute_pending_action(token="uuid-v4-token")
    MCP->>NC: PUT /remote.php/dav/files/user/Reports/summary.md
    NC-->>MCP: HTTP 201 Created
    MCP-->>Agent: {"status": "success", "bytes_written": 1420}
```

---

## 🔒 Security & Authentication

* **WebDAV Scoping:** Requests are securely scoped to authenticated user folders (`/remote.php/dav/files/{user}/`).
* **Basic Auth & App Passwords:** Uses generated Nextcloud App Passwords without exposing root account credentials.
* **HITL Guarded Actions:** Destructive operations (`write_file`, `delete_file`, `create_folder`) are protected by single-use UUID validation tokens.
* **Error Containment:** Safely catches HTTP 401/403/404, returning structured JSON error payloads to prevent agent crashes.
