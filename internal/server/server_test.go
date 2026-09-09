package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/caldav"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/deck"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/hitl"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/ocs"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/webdav"
	"github.com/mark3labs/mcp-go/mcp"
)

func setupTestServer(t *testing.T) (*Server, *httptest.Server) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/status.php":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"installed": true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/remote.php/dav/files/admin/test.txt":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`ok`))
		case r.Method == http.MethodGet && r.URL.Path == "/remote.php/dav/files/admin/test.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`file content`))
		case r.Method == http.MethodDelete && r.URL.Path == "/remote.php/dav/files/admin/test.txt":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "MKCOL" && r.URL.Path == "/remote.php/dav/files/admin/newdir":
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == "/ocs/v1.php/cloud/users/admin":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ocs":{"data":{"displayname":"Admin"}}}`))
		case r.URL.Path == "/index.php/apps/deck/api/v1.0/boards":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		case r.Method == "REPORT":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<d:multistatus></d:multistatus>`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	cfg := &config.Config{
		NCURL:    ts.URL,
		Username: "admin",
		Password: "password",
		Timeout:  5 * time.Second,
	}

	davCli := webdav.NewClient(cfg)
	caldavCli := caldav.NewClient(cfg)
	deckCli := deck.NewClient(cfg)
	ocsCli := ocs.NewClient(cfg)
	hitlMgr := hitl.NewManager(10, 5*time.Minute)

	srv := NewServer(davCli, caldavCli, deckCli, ocsCli, hitlMgr)
	return srv, ts
}

func callTool(s *Server, name string, args map[string]any) (string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	ctx := context.Background()
	var res *mcp.CallToolResult
	var err error

	switch name {
	case "execute_pending_action":
		res, err = s.handleExecutePendingAction(ctx, req)
	case "nextcloud_health":
		res, err = s.handleNextcloudHealth(ctx, req)
	case "list_files":
		res, err = s.handleListFiles(ctx, req)
	case "read_file":
		res, err = s.handleReadFile(ctx, req)
	case "write_file":
		res, err = s.handleWriteFile(ctx, req)
	case "delete_file":
		res, err = s.handleDeleteFile(ctx, req)
	case "create_folder":
		res, err = s.handleCreateFolder(ctx, req)
	case "get_user_info":
		res, err = s.handleGetUserInfo(ctx, req)
	case "list_deck_boards":
		res, err = s.handleListDeckBoards(ctx, req)
	case "list_calendar_events":
		res, err = s.handleListCalendarEvents(ctx, req)
	}

	if err != nil {
		return "", err
	}
	if len(res.Content) > 0 {
		if text, ok := res.Content[0].(mcp.TextContent); ok {
			return text.Text, nil
		}
	}
	return "", nil
}

func TestServer_HealthAndRead(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()

	// 1. Health
	healthText, err := callTool(srv, "nextcloud_health", nil)
	if err != nil {
		t.Fatalf("health tool error: %v", err)
	}
	var healthData map[string]any
	if err := json.Unmarshal([]byte(healthText), &healthData); err != nil {
		t.Fatalf("health json error: %v", err)
	}
	if healthData["status"] != "healthy" {
		t.Errorf("expected status healthy, got %v", healthData["status"])
	}

	// 2. Read File
	readText, err := callTool(srv, "read_file", map[string]any{"path": "/test.txt"})
	if err != nil {
		t.Fatalf("read_file tool error: %v", err)
	}
	var readData map[string]any
	if err := json.Unmarshal([]byte(readText), &readData); err != nil {
		t.Fatalf("read_file json error: %v", err)
	}
	if readData["status"] != "success" || readData["content"] != "file content" {
		t.Errorf("unexpected read result: %v", readData)
	}
}

func TestServer_HITL_Workflow(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()

	// 1. Trigger write_file -> returns pending_approval
	writeText, err := callTool(srv, "write_file", map[string]any{
		"path":    "/test.txt",
		"content": "new text",
	})
	if err != nil {
		t.Fatalf("write_file error: %v", err)
	}
	var writeData map[string]any
	_ = json.Unmarshal([]byte(writeText), &writeData)

	if writeData["status"] != "pending_approval" {
		t.Fatalf("expected pending_approval, got %v", writeData)
	}
	token, ok := writeData["token"].(string)
	if !ok || token == "" {
		t.Fatalf("expected valid token, got %v", writeData["token"])
	}

	// 2. Confirm via execute_pending_action
	execText, err := callTool(srv, "execute_pending_action", map[string]any{
		"token": token,
	})
	if err != nil {
		t.Fatalf("execute_pending_action error: %v", err)
	}
	var execData map[string]any
	_ = json.Unmarshal([]byte(execText), &execData)

	if execData["status"] != "success" {
		t.Errorf("expected success after execute_pending_action, got %v", execData)
	}

	// 3. Confirming same token again must fail
	execAgainText, _ := callTool(srv, "execute_pending_action", map[string]any{
		"token": token,
	})
	var againData map[string]any
	_ = json.Unmarshal([]byte(execAgainText), &againData)
	if againData["status"] != "error" {
		t.Errorf("expected error on token replay, got %v", againData)
	}
}

func TestServer_Delete_And_CreateFolder_HITL(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()

	// Delete file HITL
	delText, err := callTool(srv, "delete_file", map[string]any{"path": "/test.txt"})
	if err != nil {
		t.Fatalf("delete_file error: %v", err)
	}
	var delData map[string]any
	_ = json.Unmarshal([]byte(delText), &delData)
	delToken := delData["token"].(string)

	execDel, _ := callTool(srv, "execute_pending_action", map[string]any{"token": delToken})
	var delRes map[string]any
	_ = json.Unmarshal([]byte(execDel), &delRes)
	if delRes["status"] != "success" {
		t.Errorf("expected success for delete_file, got %v", delRes)
	}

	// Create folder HITL
	mkText, err := callTool(srv, "create_folder", map[string]any{"path": "/newdir"})
	if err != nil {
		t.Fatalf("create_folder error: %v", err)
	}
	var mkData map[string]any
	_ = json.Unmarshal([]byte(mkText), &mkData)
	mkToken := mkData["token"].(string)

	execMk, _ := callTool(srv, "execute_pending_action", map[string]any{"token": mkToken})
	var mkRes map[string]any
	_ = json.Unmarshal([]byte(execMk), &mkRes)
	if mkRes["status"] != "success" {
		t.Errorf("expected success for create_folder, got %v", mkRes)
	}
}
