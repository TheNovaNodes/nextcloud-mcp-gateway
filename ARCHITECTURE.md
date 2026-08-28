# ☁️ Architecture Documentation: Nextcloud MCP Gateway

`nextcloud-mcp-gateway` provides an asynchronous Data Plane Model Context Protocol (MCP) bridge into Nextcloud, enabling AI agents to read, write, organize, and inspect user storage and documentation.

---

## 🏗 High-Level Architecture

```mermaid
graph TD
    Agent[🤖 AI Agent / Model] -->|MCP Protocol| FastMCP[⚡ Nextcloud MCP Gateway]
    FastMCP -->|Async WebDAV / PROPFIND / PUT| WebDAV[📂 Nextcloud WebDAV API]
    FastMCP -->|CalDAV / REPORT / PUT| CalDAV[📅 Nextcloud CalDAV API]
    FastMCP -->|REST / JSON| Deck[🗃️ Nextcloud Deck API]
    FastMCP -->|REST / JSON| OCS[👥 Nextcloud OCS Cloud API]
    WebDAV -->|Port 8080 / HTTPS nc.shtab-ai.ru| Nextcloud[☁️ Nextcloud Application Server]
    CalDAV --> Nextcloud
    Deck --> Nextcloud
    OCS --> Nextcloud
    Nextcloud --> Storage[(Filesystem & Object Storage)]
    Nextcloud --> DB[(PostgreSQL Database)]
```

---

## 🔄 Sequence Workflow: File Operations

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

    Agent->>MCP: write_file(path="/Reports/summary.md", content="...")
    MCP->>NC: PUT /remote.php/dav/files/user/Reports/summary.md
    NC->>Store: Write data & update fileid
    NC-->>MCP: HTTP 201 Created
    MCP-->>Agent: {"status": "success", "bytes_written": 1420}
```

---

## 🔒 Security & Authentication

* **WebDAV Scoping:** Requests are securely scoped to authenticated user folders (`/remote.php/dav/files/{user}/`).
* **Basic Auth & App Passwords:** Uses generated Nextcloud App Passwords without exposing root account credentials.
* **Error Containment:** Safely catches HTTP 401/403/404, returning structured JSON error payloads to prevent agent crashes.
