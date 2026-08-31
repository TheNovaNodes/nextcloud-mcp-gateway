## 2026-08-31 - HTTP Connection Pooling Optimization
**Learning:** Instantiating `httpx.AsyncClient` inside a tool handler (using `async with`) prevents HTTP connection reuse, causing every API request to incur the overhead of TCP and TLS handshakes.
**Action:** Always use a single, shared global HTTP client (e.g., initialized lazily) in MCP servers that make multiple downstream API calls. This drastically reduces request latency and overhead.
