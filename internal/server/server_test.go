package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"fmt"
	"strings"
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

func callToolExtended(s *Server, name string, args map[string]any) (string, error) {
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
	case "list_deck_stacks":
		res, err = s.handleListDeckStacks(ctx, req)
	case "create_deck_card":
		res, err = s.handleCreateDeckCard(ctx, req)
	case "update_deck_card":
		res, err = s.handleUpdateDeckCard(ctx, req)
	case "delete_deck_card":
		res, err = s.handleDeleteDeckCard(ctx, req)
	case "list_calendar_events":
		res, err = s.handleListCalendarEvents(ctx, req)
	case "create_calendar_event":
		res, err = s.handleCreateCalendarEvent(ctx, req)
	case "delete_calendar_event":
		res, err = s.handleDeleteCalendarEvent(ctx, req)
	}

	if err != nil {
		return "", err
	}
	if res == nil {
	    return "", fmt.Errorf("res is nil for %s", name)
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
	healthText, err := callToolExtended(srv, "nextcloud_health", nil)
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
	readText, err := callToolExtended(srv, "read_file", map[string]any{"path": "/test.txt"})
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
	writeText, err := callToolExtended(srv, "write_file", map[string]any{
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
	execText, err := callToolExtended(srv, "execute_pending_action", map[string]any{
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
	execAgainText, _ := callToolExtended(srv, "execute_pending_action", map[string]any{
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
	delText, err := callToolExtended(srv, "delete_file", map[string]any{"path": "/test.txt"})
	if err != nil {
		t.Fatalf("delete_file error: %v", err)
	}
	var delData map[string]any
	_ = json.Unmarshal([]byte(delText), &delData)
	delToken := delData["token"].(string)

	execDel, _ := callToolExtended(srv, "execute_pending_action", map[string]any{"token": delToken})
	var delRes map[string]any
	_ = json.Unmarshal([]byte(execDel), &delRes)
	if delRes["status"] != "success" {
		t.Errorf("expected success for delete_file, got %v", delRes)
	}

	// Create folder HITL
	mkText, err := callToolExtended(srv, "create_folder", map[string]any{"path": "/newdir"})
	if err != nil {
		t.Fatalf("create_folder error: %v", err)
	}
	var mkData map[string]any
	_ = json.Unmarshal([]byte(mkText), &mkData)
	mkToken := mkData["token"].(string)

	execMk, _ := callToolExtended(srv, "execute_pending_action", map[string]any{"token": mkToken})
	var mkRes map[string]any
	_ = json.Unmarshal([]byte(execMk), &mkRes)
	if mkRes["status"] != "success" {
		t.Errorf("expected success for create_folder, got %v", mkRes)
	}
}

func TestServer_UncoveredHandlers(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()

	tests := []struct {
		toolName string
		args     map[string]any
		wantSub  string
	}{
		{"list_files", map[string]any{"path": "/", "offset": 0, "limit": 50}, `"status":"success"`},
		{"get_user_info", nil, `"status":"success"`},
		{"list_deck_boards", nil, `"status":"success"`},
		{"list_deck_stacks", map[string]any{"board_id": 1}, `"status":"success"`},
		{"create_deck_card", map[string]any{"board_id": 1, "stack_id": 1, "title": "t", "description": "d"}, `"status":"success"`}, // mock doesn't handle POST well but we just want coverage of handler
		{"update_deck_card", map[string]any{"board_id": 1, "stack_id": 1, "card_id": 1, "title": "t", "description": "d", "order": 0}, `"status":"success"`},
		{"delete_deck_card", map[string]any{"board_id": 1, "stack_id": 1, "card_id": 1}, `"status":"success"`},
		{"list_calendar_events", map[string]any{"calendar_name": "personal"}, `"status":"success"`},
		{"create_calendar_event", map[string]any{"event_uid": "uid", "summary": "sum", "dtstart": "start", "dtend": "end", "calendar_name": "personal"}, `"status":"success"`},
		{"delete_calendar_event", map[string]any{"event_uid": "uid", "calendar_name": "personal"}, `"status":"success"`},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			resText, err := callToolExtended(srv, tt.toolName, tt.args)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.toolName, err)
			}
			if !strings.Contains(resText, tt.wantSub) {
				t.Errorf("tool %s expected response containing %q, got: %s", tt.toolName, tt.wantSub, resText)
			}
		})
	}
}

// Ensure the jsonResult failure path is covered (e.g. chan is not marshallable)
func TestServer_jsonResult_Error(t *testing.T) {
	res, err := jsonResult(make(chan int))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Errorf("expected IsError true")
	}
	if text, ok := res.Content[0].(mcp.TextContent); !ok || !strings.Contains(text.Text, "JSON serialization error") {
		t.Errorf("expected JSON serialization error content")
	}
}

func TestServer_MCPServer_Coverage(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()

	if srv.MCPServer() == nil {
		t.Errorf("expected MCPServer to be returned")
	}
}

func TestServer_ExecutePendingAction_UnknownType(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	
	// Inject a fake unknown action
	token := srv.hitlMgr.Request("unknown_action_type", nil)["token"].(string)
	
	resText, err := callToolExtended(srv, "execute_pending_action", map[string]any{"token": token})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(resText, "Unknown action type") {
		t.Errorf("expected Unknown action type error, got %s", resText)
	}
}

func TestServer_ExecutePendingAction_NoToken(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	
	resText, err := callToolExtended(srv, "execute_pending_action", map[string]any{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(resText, "Token is required") {
		t.Errorf("expected Token is required error, got %s", resText)
	}
}
