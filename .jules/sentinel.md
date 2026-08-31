## 2026-08-31 - [Insecure TLS Configuration]
**Vulnerability:** Found `verify=False` hardcoded in all `httpx.AsyncClient` initializations, which disables SSL certificate verification, making the gateway vulnerable to Man-in-the-Middle (MITM) attacks.
**Learning:** This is a common but dangerous pattern often introduced during development (e.g. testing with self-signed certificates) and inadvertently left in production code. It silently compromises the security of all communication with the Nextcloud server.
**Prevention:** Introduce a configurable `verify_ssl` flag in the configuration (e.g., `NC_VERIFY_SSL=true`) that is true by default. Only allow disabling it explicitly via environment variables. Use `verify=config.verify_ssl` instead of hardcoding `verify=False`.
