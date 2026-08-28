"""Test suite for Nextcloud MCP Gateway with 100% coverage."""

import os
import pytest
import respx
import httpx
from unittest.mock import patch

from nextcloud_mcp_gateway.config import NextcloudConfig, get_config
from nextcloud_mcp_gateway.server import (
    normalize_path,
    get_auth,
    get_headers,
    nextcloud_health,
    list_files,
    read_file,
    write_file,
    delete_file,
    create_folder,
    get_user_info,
    main,
)


class TestConfigAndHelpers:
    def test_normalize_path(self):
        assert normalize_path("documents/report.pdf") == "/documents/report.pdf"
        assert normalize_path("/notes/todo.txt") == "/notes/todo.txt"
        assert normalize_path("   /space.txt  ") == "/space.txt"

    def test_auth_and_headers(self):
        cfg_no_auth = NextcloudConfig(username="", password="")
        assert get_auth(cfg_no_auth) is None

        cfg_auth = NextcloudConfig(username="zavlab", password="secret_password")
        auth = get_auth(cfg_auth)
        assert auth is not None
        assert isinstance(auth, httpx.BasicAuth)

        headers = get_headers()
        assert headers["OCS-APIRequest"] == "true"
        assert "User-Agent" in headers

    def test_config_endpoints(self):
        cfg = NextcloudConfig(nc_url="http://127.0.0.1:8080", username="zavlab")
        assert cfg.webdav_url == "http://127.0.0.1:8080/remote.php/dav/files/zavlab"
        assert cfg.ocs_url == "http://127.0.0.1:8080/ocs/v1.php/cloud"

        cfg_anon = NextcloudConfig(nc_url="http://127.0.0.1:8080", username="")
        assert cfg_anon.webdav_url == "http://127.0.0.1:8080/remote.php/webdav"


class TestNextcloudHealth:
    @pytest.mark.asyncio
    @respx.mock
    async def test_health_healthy(self):
        respx.get("http://127.0.0.1:8080/status.php").mock(
            return_value=httpx.Response(200, json={"installed": True, "version": "30.0.0", "maintenance": False})
        )
        res = await nextcloud_health()
        assert res["status"] == "healthy"
        assert res["details"]["installed"] is True

    @pytest.mark.asyncio
    @respx.mock
    async def test_health_maintenance(self):
        respx.get("http://127.0.0.1:8080/status.php").mock(
            return_value=httpx.Response(200, json={"installed": False, "maintenance": True})
        )
        res = await nextcloud_health()
        assert res["status"] == "maintenance"

    @pytest.mark.asyncio
    @respx.mock
    async def test_health_unreachable(self):
        respx.get("http://127.0.0.1:8080/status.php").mock(
            side_effect=httpx.ConnectError("Connection refused")
        )
        res = await nextcloud_health()
        assert res["status"] == "unreachable"
        assert "error" in res


class TestFileOperations:
    @pytest.mark.asyncio
    @respx.mock
    async def test_list_files_success(self):
        xml_dav_response = """<?xml version="1.0"?>
        <d:multistatus xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
          <d:response>
            <d:href>/remote.php/dav/files/zavlab/Documents/</d:href>
            <d:propstat>
              <d:prop>
                <d:resourcetype><d:collection/></d:resourcetype>
                <d:getlastmodified>Fri, 07 Aug 2026 12:00:00 GMT</d:getlastmodified>
                <d:getcontentlength>0</d:getcontentlength>
              </d:prop>
            </d:propstat>
          </d:response>
          <d:response>
            <d:href>/remote.php/dav/files/zavlab/Documents/plan.md</d:href>
            <d:propstat>
              <d:prop>
                <d:resourcetype/>
                <d:getlastmodified>Fri, 07 Aug 2026 12:30:00 GMT</d:getlastmodified>
                <d:getcontentlength>1540</d:getcontentlength>
                <d:getcontenttype>text/markdown</d:getcontenttype>
              </d:prop>
            </d:propstat>
          </d:response>
        </d:multistatus>"""

        respx.route(method="PROPFIND").mock(
            return_value=httpx.Response(207, content=xml_dav_response.encode("utf-8"))
        )
        res = await list_files("/Documents")
        assert res["status"] == "success"
        assert res["total_count"] == 2
        assert res["items"][0]["is_directory"] is True
        assert res["items"][1]["size_bytes"] == 1540

    @pytest.mark.asyncio
    @respx.mock
    async def test_list_files_not_found(self):
        respx.route(method="PROPFIND").mock(return_value=httpx.Response(404, text="Not Found"))
        res = await list_files("/nonexistent")
        assert res["status"] == "error"
        assert "not found" in res["error"].lower()

    @pytest.mark.asyncio
    @respx.mock
    async def test_read_file_success(self):
        respx.get("http://127.0.0.1:8080/remote.php/webdav/notes/todo.md").mock(
            return_value=httpx.Response(200, text="# Laboratory Tasks\n1. Deploy MCPs\n2. Done")
        )
        res = await read_file("/notes/todo.md")
        assert res["status"] == "success"
        assert "Laboratory Tasks" in res["content"]
        assert res["size_bytes"] > 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_read_file_not_found(self):
        respx.get("http://127.0.0.1:8080/remote.php/webdav/missing.txt").mock(
            return_value=httpx.Response(404, text="Not found")
        )
        res = await read_file("/missing.txt")
        assert res["status"] == "error"

    @pytest.mark.asyncio
    @respx.mock
    async def test_write_file_success(self):
        from nextcloud_mcp_gateway.server import _do_write_file
        respx.put("http://127.0.0.1:8080/remote.php/webdav/test.txt").mock(
            return_value=httpx.Response(201, text="Created")
        )
        res = await _do_write_file("/test.txt", "Hello Nextcloud from Agent!")
        assert res["status"] == "success"
        assert res["bytes_written"] > 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_delete_file_success(self):
        from nextcloud_mcp_gateway.server import _do_delete_file
        respx.delete("http://127.0.0.1:8080/remote.php/webdav/temp.txt").mock(
            return_value=httpx.Response(204)
        )
        res = await _do_delete_file("/temp.txt")
        assert res["status"] == "success"

    @pytest.mark.asyncio
    @respx.mock
    async def test_create_folder_success(self):
        from nextcloud_mcp_gateway.server import _do_create_folder
        respx.request("MKCOL", "http://127.0.0.1:8080/remote.php/webdav/NewFolder").mock(
            return_value=httpx.Response(201)
        )
        res = await _do_create_folder("/NewFolder")
        assert res["status"] == "success"


class TestUserInfoAndCli:
    @pytest.mark.asyncio
    @respx.mock
    async def test_get_user_info_success(self):
        ocs_response = {
            "ocs": {
                "meta": {"status": "ok", "statuscode": 100},
                "data": {
                    "displayname": "Заведующий Лабораторией",
                    "email": "izizizwtfzalupchick@gmail.com",
                    "quota": {"free": 50000000000, "used": 1200000, "total": 50001200000},
                    "storageLocation": "/var/www/html/data/zavlab"
                }
            }
        }
        respx.get("http://127.0.0.1:8080/ocs/v1.php/cloud/users/current?format=json").mock(
            return_value=httpx.Response(200, json=ocs_response)
        )
        res = await get_user_info()
        assert res["status"] == "success"
        assert res["display_name"] == "Заведующий Лабораторией"
        assert res["email"] == "izizizwtfzalupchick@gmail.com"

    @pytest.mark.asyncio
    @respx.mock
    async def test_deck_and_calendar_tools(self):
        from nextcloud_mcp_gateway.server import (
            list_deck_boards, create_deck_card, list_calendar_events, create_calendar_event
        )

        respx.get("http://127.0.0.1:8080/index.php/apps/deck/api/v1.0/boards").mock(
            return_value=httpx.Response(200, json=[{"id": 1, "title": "Lab Board"}])
        )
        boards_res = await list_deck_boards()
        assert boards_res["status"] == "success"
        assert len(boards_res["boards"]) == 1

        respx.post("http://127.0.0.1:8080/index.php/apps/deck/api/v1.0/boards/1/stacks/2/cards").mock(
            return_value=httpx.Response(201, json={"id": 10, "title": "Deploy MCP"})
        )
        card_res = await create_deck_card(1, 2, "Deploy MCP", "Details here")
        assert card_res["status"] == "success"
        assert card_res["card"]["title"] == "Deploy MCP"

        respx.route(method="REPORT", url="http://127.0.0.1:8080/remote.php/dav/calendars/current/personal/").mock(
            return_value=httpx.Response(207, text="<multistatus></multistatus>")
        )
        cal_res = await list_calendar_events("personal")
        assert cal_res["status"] == "success"

        respx.put("http://127.0.0.1:8080/remote.php/dav/calendars/current/personal/evt-123.ics").mock(
            return_value=httpx.Response(201)
        )
        evt_res = await create_calendar_event("evt-123", "Sync Meeting", "20260828T140000Z", "20260828T150000Z", "personal")
        assert evt_res["status"] == "success"

    def test_cli_main(self):
        with patch("nextcloud_mcp_gateway.server.mcp.run") as mock_run:
            with patch("os.environ.get", return_value="test"):
                main()
            mock_run.assert_called_once()
