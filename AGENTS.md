Part 1: TheNovaNodes Core Invariants (Universal Standard)
1. Strict Git Flow (ПРАВИЛА КРОВИ):
   - NEVER push directly to main or master branches.
   - All changes must go through dedicated branches (feat/..., fix/..., docs/..., ci/...) and Pull Requests.
   - NEVER merge PRs without explicit approval from ЗавЛаб.
   - No force-push on upstream branches.
2. Security & Credential Hygiene:
   - NEVER hardcode or log passwords, tokens, or credentials (especially NC_APP_PASSWORD, NC_USER).
   - All credentials must be loaded dynamically from environment variables.
3. Deadlock & Timeout Guardrails:
   - All network calls (WebDAV, CalDAV, Deck, OCS) must have explicit timeouts (configured via config.Timeout).
   - Auxiliary commands must use hard timeouts.
   - MCP stdio servers must redirect stdin (< /dev/null) during smoke tests.
   - Never loop endlessly without bounds.
4. Continuous Verification:
   - Never report a task complete without running local verification commands.
   - Use native project tools directly (go vet, go test -v -race, make build).

Part 2: Repository Profile & Specific Directives (nextcloud-mcp-gateway)
1. Project Overview & Tech Stack:
   - Go 1.22+ / 1.25 compatible.
   - Dependencies: github.com/mark3labs/mcp-go.
   - Packages: internal/config, internal/hitl, internal/webdav, internal/caldav, internal/deck, internal/ocs, internal/server.
   - Entry point: cmd/nextcloud-mcp-gateway/main.go (starts stdio MCP server via mcpserver.ServeStdio).
2. The Golden Loop (Mandatory Verification Commands):
   - go vet ./...
   - go test -v -race ./...
   - make build
3. Architectural Invariants & Taboos:
   - CRITICAL HITL (Human-in-the-Loop) INVARIANT: All destructive actions (write_file, delete_file, create_folder) are strictly guarded by internal/hitl.Manager. Agents are FORBIDDEN from bypassing or relaxing HITL token checks!
   - WebDAV Memory Ceiling: ReadFile enforces a 10MB limit via io.LimitReader to prevent Out-Of-Memory crashes on large files.
   - Connection / Socket Hygiene: HTTP response bodies must always be drained (e.g. io.CopyN(io.Discard, resp.Body, 512)) before Close() to preserve HTTP Keep-Alive connection pooling.
   - Path Traversal Guard: internal/webdav/path.go strictly enforces path normalization and blocks directory traversal (..).
   - Stdio Protocol Hygiene: os.Stdout is exclusively reserved for MCP JSON-RPC protocol. All application logs must go to os.Stderr.
   - Non-destructive read tools (list_files, read_file, list_calendar_events, list_deck_boards, list_deck_stacks, nextcloud_health, get_user_info) execute directly without HITL tokens.
4. PR & Commit Conventions:
   - Conventional Commits (feat(...): ..., fix(...): ..., ci(...): ..., docs(...): ...).
