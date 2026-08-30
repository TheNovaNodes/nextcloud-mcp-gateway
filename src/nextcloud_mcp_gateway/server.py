"""Nextcloud Data Plane FastMCP Server for AI Agent interaction."""

import sys
import xml.etree.ElementTree as ET
from typing import Any, Dict, List, Optional
import httpx
import uuid

try:
    from mcp.server.fastmcp import FastMCP
except ImportError:
    try:
        from fastmcp import FastMCP
    except ImportError:
        class FastMCP:  # type: ignore
            def __init__(self, name: str, **kwargs: Any) -> None:
                self.name = name
                self.tools: Dict[str, Any] = {}

            def tool(self) -> Any:
                def decorator(fn: Any) -> Any:
                    self.tools[fn.__name__] = fn
                    return fn
                return decorator

            def run(self, transport: str = "stdio") -> None:
                pass

from nextcloud_mcp_gateway.config import get_config, NextcloudConfig

# Initialize FastMCP Server instance
mcp = FastMCP("nextcloud-mcp-gateway")


PENDING_ACTIONS: Dict[str, Dict[str, Any]] = {}

def request_hitl(action_type: str, details: Dict[str, Any]) -> Dict[str, str]:
    token = str(uuid.uuid4())
    PENDING_ACTIONS[token] = {"type": action_type, "details": details}
    return {
        "status": "pending_approval", 
        "message": f"🚨 HITL REQUIRED: Action '{action_type}' is blocked. To confirm, use 'execute_pending_action' with token.",
        "token": token
    }

@mcp.tool()
async def execute_pending_action(token: str) -> Dict[str, Any]:
    """Execute a destructive action that was previously blocked by HITL."""
    if token not in PENDING_ACTIONS:
        return {"status": "error", "error": "Invalid, expired, or already executed HITL token."}
        
    action = PENDING_ACTIONS.pop(token)
    
    # We route the execution to the inner functions
    if action["type"] == "delete_file":
        return await _do_delete_file(action["details"]["path"])
    elif action["type"] == "write_file":
        return await _do_write_file(action["details"]["path"], action["details"]["content"])
    elif action["type"] == "create_folder":
        return await _do_create_folder(action["details"]["path"])
    else:
        return {"status": "error", "error": f"Unknown action type: {action['type']}"}



def get_auth(config: NextcloudConfig) -> Optional[httpx.BasicAuth]:
    """Provide HTTP Basic Auth credentials if present."""
    if config.username and config.password:
        return httpx.BasicAuth(config.username, config.password)
    return None


def get_headers() -> Dict[str, str]:
    """Default headers for Nextcloud API and WebDAV interaction."""
    return {
        "OCS-APIRequest": "true",
        "User-Agent": "TheNovaNodes-Nextcloud-MCP-Gateway/1.0",
        "Accept": "application/json, text/xml, */*"
    }


def normalize_path(path: str) -> str:
    """Ensure path is clean for WebDAV requests."""
    p = path.strip()
    if not p.startswith("/"):
        p = "/" + p
    return p


@mcp.tool()
async def nextcloud_health() -> Dict[str, Any]:
    """Check Nextcloud instance health, version, status.php and WebDAV availability."""
    config = get_config()
    status_url = f"{config.nc_url}/status.php"
    
    health_report: Dict[str, Any] = {
        "status": "unknown",
        "endpoint": config.nc_url,
        "public_url": config.public_url,
        "authenticated": bool(config.username and config.password),
        "user": config.username or "anonymous",
        "details": {}
    }

    try:
        async with httpx.AsyncClient(timeout=config.timeout, verify=False) as client:
            resp = await client.get(status_url, headers=get_headers())
            if resp.status_code == 200:
                try:
                    data = resp.json()
                    health_report["status"] = "healthy" if data.get("installed") else "maintenance"
                    health_report["details"] = data
                except Exception:
                    health_report["status"] = "healthy"
                    health_report["details"] = {"raw": resp.text[:200]}
            else:
                health_report["status"] = "degraded"
                health_report["error"] = f"HTTP {resp.status_code}: {resp.text[:200]}"
    except Exception as e:
        health_report["status"] = "unreachable"
        health_report["error"] = str(e)

    return health_report


@mcp.tool()
async def list_files(path: str = "/", offset: int = 0, limit: int = 50) -> Dict[str, Any]:
    """List files and folders in a Nextcloud directory via WebDAV PROPFIND."""
    config = get_config()
    clean_p = normalize_path(path)
    target_url = f"{config.webdav_url}{clean_p}"
    
    headers = get_headers()
    headers["Depth"] = "1"
    headers["Content-Type"] = "application/xml; charset=utf-8"

    propfind_body = """<?xml version="1.0" encoding="utf-8" ?>
    <d:propfind xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns" xmlns:nc="http://nextcloud.org/ns">
      <d:prop>
        <d:getlastmodified/>
        <d:getcontentlength/>
        <d:getcontenttype/>
        <d:resourcetype/>
        <oc:fileid/>
        <oc:size/>
      </d:prop>
    </d:propfind>"""

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.request("PROPFIND", target_url, headers=headers, content=propfind_body)
            
            if resp.status_code in [207, 200]:
                items = []
                try:
                    root = ET.fromstring(resp.content)
                    ns = {"d": "DAV:", "oc": "http://owncloud.org/ns"}
                    for response in root.findall("d:response", ns):
                        href = response.findtext("d:href", "", ns)
                        propstat = response.find("d:propstat", ns)
                        if propstat is not None:
                            prop = propstat.find("d:prop", ns)
                            if prop is not None:
                                is_dir = prop.find("d:resourcetype/d:collection", ns) is not None
                                size = prop.findtext("d:getcontentlength", "0", ns)
                                mod = prop.findtext("d:getlastmodified", "", ns)
                                content_type = prop.findtext("d:getcontenttype", "directory" if is_dir else "file", ns)
                                items.append({
                                    "href": href,
                                    "is_directory": is_dir,
                                    "size_bytes": int(size) if size.isdigit() else 0,
                                    "last_modified": mod,
                                    "content_type": content_type
                                })

                except Exception as ex:
                    return {"status": "success", "raw_xml": resp.text[:1000], "parse_error": str(ex)}

                safe_limit = min(limit, 100)
                paginated_items = items[offset:offset+safe_limit]
                return {
                    "status": "success", 
                    "path": clean_p, 
                    "total_count": len(items), 
                    "returned_count": len(paginated_items),
                    "offset": offset,
                    "limit": safe_limit,
                    "items": paginated_items
                }

            elif resp.status_code == 404:
                return {"status": "error", "error": f"Path not found: {clean_p}"}
            elif resp.status_code in [401, 403]:
                return {"status": "error", "error": "Authentication failed. Provide valid NC_USER and NC_APP_PASSWORD."}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


@mcp.tool()
async def read_file(path: str) -> Dict[str, Any]:
    """Read textual content of a file from Nextcloud storage."""
    config = get_config()
    clean_p = normalize_path(path)
    target_url = f"{config.webdav_url}{clean_p}"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.get(target_url, headers=get_headers())
            if resp.status_code == 200:
                return {
                    "status": "success",
                    "path": clean_p,
                    "size_bytes": len(resp.content),
                    "content": resp.text
                }
            elif resp.status_code == 404:
                return {"status": "error", "error": f"File not found: {clean_p}"}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


async def _do_write_file(path: str, content: str) -> Dict[str, Any]:
    """Create or overwrite a file in Nextcloud storage via WebDAV PUT."""
    config = get_config()
    clean_p = normalize_path(path)
    target_url = f"{config.webdav_url}{clean_p}"

    headers = get_headers()
    headers["Content-Type"] = "text/plain; charset=utf-8"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.put(target_url, headers=headers, content=content.encode("utf-8"))
            if resp.status_code in [200, 201, 204]:
                return {
                    "status": "success",
                    "path": clean_p,
                    "bytes_written": len(content.encode("utf-8")),
                    "message": "File written successfully"
                }
            elif resp.status_code in [401, 403]:
                return {"status": "error", "error": "Unauthorized to write file in Nextcloud."}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


async def _do_delete_file(path: str) -> Dict[str, Any]:
    """Delete a file or folder from Nextcloud storage via WebDAV DELETE."""
    config = get_config()
    clean_p = normalize_path(path)
    target_url = f"{config.webdav_url}{clean_p}"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.delete(target_url, headers=get_headers())
            if resp.status_code in [200, 204]:
                return {"status": "success", "path": clean_p, "message": "Resource deleted"}
            elif resp.status_code == 404:
                return {"status": "error", "error": f"Resource not found: {clean_p}"}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


async def _do_create_folder(path: str) -> Dict[str, Any]:
    """Create a new folder in Nextcloud storage via WebDAV MKCOL."""
    config = get_config()
    clean_p = normalize_path(path)
    target_url = f"{config.webdav_url}{clean_p}"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.request("MKCOL", target_url, headers=get_headers())
            if resp.status_code in [201, 200]:
                return {"status": "success", "path": clean_p, "message": "Folder created successfully"}
            elif resp.status_code == 405:
                return {"status": "exists", "path": clean_p, "message": "Folder already exists"}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


@mcp.tool()
async def get_user_info() -> Dict[str, Any]:
    """Retrieve user storage quota, display name, and details via Nextcloud OCS API."""
    config = get_config()
    user = config.username or "current"
    target_url = f"{config.ocs_url}/users/{user}?format=json"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.get(target_url, headers=get_headers())
            if resp.status_code == 200:
                data = resp.json()
                ocs_data = data.get("ocs", {}).get("data", {})
                return {
                    "status": "success",
                    "user": user,
                    "display_name": ocs_data.get("displayname"),
                    "email": ocs_data.get("email"),
                    "quota": ocs_data.get("quota", {}),
                    "storage_location": ocs_data.get("storageLocation")
                }
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}



@mcp.tool()

async def list_deck_boards() -> Dict[str, Any]:
    """List all Nextcloud Deck Kanban boards available to the user."""
    config = get_config()
    target_url = f"{config.nc_url}/index.php/apps/deck/api/v1.0/boards"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.get(target_url, headers=get_headers())
            if resp.status_code == 200:
                return {"status": "success", "boards": resp.json()}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


@mcp.tool()
async def create_deck_card(board_id: int, stack_id: int, title: str, description: str = "") -> Dict[str, Any]:
    """Create a new Kanban card in Nextcloud Deck."""
    config = get_config()
    target_url = f"{config.nc_url}/index.php/apps/deck/api/v1.0/boards/{board_id}/stacks/{stack_id}/cards"
    payload = {"title": title, "description": description, "type": "plain"}

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.post(target_url, headers=get_headers(), json=payload)
            if resp.status_code in [200, 201]:
                return {"status": "success", "card": resp.json()}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


@mcp.tool()
async def list_calendar_events(calendar_name: str = "personal") -> Dict[str, Any]:
    """List events from a Nextcloud CalDAV calendar."""
    config = get_config()
    user = config.username or "current"
    target_url = f"{config.nc_url}/remote.php/dav/calendars/{user}/{calendar_name}/"

    headers = get_headers()
    headers["Depth"] = "1"
    headers["Content-Type"] = "application/xml; charset=utf-8"

    propfind_body = """<?xml version="1.0" encoding="utf-8" ?>
    <c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
      <d:prop><d:getetag/><c:calendar-data/></d:prop>
      <c:filter><c:comp-filter name="VCALENDAR"/></c:filter>
    </c:calendar-query>"""

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.request("REPORT", target_url, headers=headers, content=propfind_body)
            if resp.status_code in [207, 200]:
                return {"status": "success", "raw_caldav_xml": resp.text[:2000]}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}


@mcp.tool()
async def create_calendar_event(event_uid: str, summary: str, dtstart: str, dtend: str, calendar_name: str = "personal") -> Dict[str, Any]:
    """Create a new event in Nextcloud CalDAV calendar using iCalendar (.ics)."""
    config = get_config()
    user = config.username or "current"
    target_url = f"{config.nc_url}/remote.php/dav/calendars/{user}/{calendar_name}/{event_uid}.ics"

    ics_content = f"""BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//TheNovaNodes//Kairos Calendar//EN
BEGIN:VEVENT
UID:{event_uid}
SUMMARY:{summary}
DTSTART:{dtstart}
DTEND:{dtend}
END:VEVENT
END:VCALENDAR"""

    headers = get_headers()
    headers["Content-Type"] = "text/calendar; charset=utf-8"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.put(target_url, headers=headers, content=ics_content.encode("utf-8"))
            if resp.status_code in [200, 201, 204]:
                return {"status": "success", "event_uid": event_uid, "message": "Event created"}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}



@mcp.tool()
async def list_deck_stacks(board_id: int) -> Dict[str, Any]:
    """List all stacks (columns) in a Nextcloud Deck board."""
    config = get_config()
    target_url = f"{config.nc_url}/index.php/apps/deck/api/v1.0/boards/{board_id}/stacks"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.get(target_url, headers=get_headers())
            if resp.status_code == 200:
                return {"status": "success", "stacks": resp.json()}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}

@mcp.tool()
async def update_deck_card(board_id: int, stack_id: int, card_id: int, title: str, description: str = "", order: int = 0) -> Dict[str, Any]:
    """Update an existing Kanban card in Nextcloud Deck (e.g. to move it to another stack/column)."""
    config = get_config()
    target_url = f"{config.nc_url}/index.php/apps/deck/api/v1.0/boards/{board_id}/stacks/{stack_id}/cards/{card_id}"
    user = config.username or "admin"
    payload = {"title": title, "description": description, "order": order, "stackId": stack_id, "type": "plain", "owner": user}

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.put(target_url, headers=get_headers(), json=payload)
            if resp.status_code == 200:
                return {"status": "success", "card": resp.json()}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}

@mcp.tool()
async def delete_deck_card(board_id: int, stack_id: int, card_id: int) -> Dict[str, Any]:
    """Delete a Kanban card in Nextcloud Deck."""
    config = get_config()
    target_url = f"{config.nc_url}/index.php/apps/deck/api/v1.0/boards/{board_id}/stacks/{stack_id}/cards/{card_id}"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.delete(target_url, headers=get_headers())
            if resp.status_code == 200:
                return {"status": "success", "message": "Card deleted"}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}

@mcp.tool()
async def delete_calendar_event(event_uid: str, calendar_name: str = "personal") -> Dict[str, Any]:
    """Delete an event from a Nextcloud CalDAV calendar."""
    config = get_config()
    user = config.username or "current"
    target_url = f"{config.nc_url}/remote.php/dav/calendars/{user}/{calendar_name}/{event_uid}.ics"

    try:
        async with httpx.AsyncClient(auth=get_auth(config), timeout=config.timeout, verify=False) as client:
            resp = await client.delete(target_url, headers=get_headers())
            if resp.status_code in [200, 204]:
                return {"status": "success", "message": "Event deleted"}
            elif resp.status_code == 404:
                return {"status": "error", "error": "Event not found"}
            else:
                return {"status": "error", "code": resp.status_code, "error": resp.text[:300]}
    except Exception as e:
        return {"status": "error", "error": str(e)}

@mcp.tool()
async def write_file(path: str, content: str) -> Dict[str, Any]:
    """Create or overwrite a file in Nextcloud storage. (HITL protected)"""
    return request_hitl("write_file", {"path": path, "content": content})

@mcp.tool()
async def delete_file(path: str) -> Dict[str, Any]:
    """Delete a file or folder from Nextcloud storage. (HITL protected)"""
    return request_hitl("delete_file", {"path": path})

@mcp.tool()
async def create_folder(path: str) -> Dict[str, Any]:
    """Create a new folder in Nextcloud storage. (HITL protected)"""
    return request_hitl("create_folder", {"path": path})

def main() -> None:
    """Server CLI entrypoint."""
    _config = get_config()
    if (not _config.username or not _config.password) and "pytest" not in sys.modules:
        print("❌ CRITICAL: Nextcloud credentials (NC_USER, NC_APP_PASSWORD) missing in Vault/Env. Aborting.", file=sys.stderr)
        sys.exit(1)
    mcp.run()


if __name__ == "__main__":
    main()
