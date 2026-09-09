package deck_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/deck"
)

func TestClient_Deck(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 1, "title": "Main Board"}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 10, "title": "Todo"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks/10/cards":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": 100, "title": "New Task"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks/10/cards/100":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "title": "Updated Task"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks/10/cards/100":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	cfg := &config.Config{
		NCURL:    ts.URL,
		Username: "admin",
		Password: "password",
		Timeout:  5 * time.Second,
	}
	cli := deck.NewClient(cfg)
	ctx := context.Background()

	// List boards
	boardsRes, err := cli.ListBoards(ctx)
	if err != nil || boardsRes["status"] != "success" {
		t.Fatalf("unexpected boards result: %v, err: %v", boardsRes, err)
	}

	// List stacks
	stacksRes, err := cli.ListStacks(ctx, 1)
	if err != nil || stacksRes["status"] != "success" {
		t.Fatalf("unexpected stacks result: %v, err: %v", stacksRes, err)
	}

	// Create card
	createRes, err := cli.CreateCard(ctx, 1, 10, "New Task", "Desc")
	if err != nil || createRes["status"] != "success" {
		t.Fatalf("unexpected create card result: %v, err: %v", createRes, err)
	}

	// Update card
	updateRes, err := cli.UpdateCard(ctx, 1, 10, 100, "Updated Task", "Desc", 1)
	if err != nil || updateRes["status"] != "success" {
		t.Fatalf("unexpected update card result: %v, err: %v", updateRes, err)
	}

	// Delete card
	delRes, err := cli.DeleteCard(ctx, 1, 10, 100)
	if err != nil || delRes["status"] != "success" {
		t.Fatalf("unexpected delete card result: %v, err: %v", delRes, err)
	}
}

func TestClient_ListBoards_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
		wantErrSub string
	}{
		{"Unauthorized", http.StatusUnauthorized, "Auth required", "error", "Auth required"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "error", "Server Error"},
		{"Invalid JSON", http.StatusOK, "{invalid}", "error", "{invalid}"},
		{"Network Error", 0, "", "error", "Get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.statusCode == 0 {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				}))
				ts.Close()
			} else {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer ts.Close()
			}

			cfg := &config.Config{
				NCURL:    ts.URL,
				Timeout:  1 * time.Second,
			}
			cli := deck.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.ListBoards(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
			if tt.wantErrSub != "" && res["status"] == "error" && tt.statusCode != 0 {
				errStr, _ := res["error"].(string)
				if !strings.Contains(errStr, tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrSub, errStr)
				}
			}
		})
	}
}

func TestClient_ListStacks_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
		wantErrSub string
	}{
		{"Not Found", http.StatusNotFound, "Not Found", "error", "Not Found"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "error", "Server Error"},
		{"Invalid JSON", http.StatusOK, "{invalid}", "error", "{invalid}"},
		{"Network Error", 0, "", "error", "Get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.statusCode == 0 {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				}))
				ts.Close()
			} else {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer ts.Close()
			}

			cfg := &config.Config{
				NCURL:    ts.URL,
				Timeout:  1 * time.Second,
			}
			cli := deck.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.ListStacks(ctx, 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
			if tt.wantErrSub != "" && res["status"] == "error" && tt.statusCode != 0 {
				errStr, _ := res["error"].(string)
				if !strings.Contains(errStr, tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrSub, errStr)
				}
			}
		})
	}
}

func TestClient_CreateCard_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
		wantErrSub string
	}{
		{"Forbidden", http.StatusForbidden, "Forbidden", "error", "Forbidden"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "error", "Server Error"},
		{"Invalid JSON", http.StatusCreated, "{invalid}", "error", "{invalid}"},
		{"Network Error", 0, "", "error", "Post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.statusCode == 0 {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				}))
				ts.Close()
			} else {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer ts.Close()
			}

			cfg := &config.Config{
				NCURL:    ts.URL,
				Timeout:  1 * time.Second,
			}
			cli := deck.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.CreateCard(ctx, 1, 2, "Title", "Desc")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
			if tt.wantErrSub != "" && res["status"] == "error" && tt.statusCode != 0 {
				errStr, _ := res["error"].(string)
				if !strings.Contains(errStr, tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrSub, errStr)
				}
			}
		})
	}
}

func TestClient_UpdateCard_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
		wantErrSub string
	}{
		{"Forbidden", http.StatusForbidden, "Forbidden", "error", "Forbidden"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "error", "Server Error"},
		{"Invalid JSON", http.StatusOK, "{invalid}", "error", "{invalid}"},
		{"Network Error", 0, "", "error", "Put"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.statusCode == 0 {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				}))
				ts.Close()
			} else {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer ts.Close()
			}

			cfg := &config.Config{
				NCURL:    ts.URL,
				Timeout:  1 * time.Second,
			}
			cli := deck.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.UpdateCard(ctx, 1, 2, 3, "Title", "Desc", 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
			if tt.wantErrSub != "" && res["status"] == "error" && tt.statusCode != 0 {
				errStr, _ := res["error"].(string)
				if !strings.Contains(errStr, tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrSub, errStr)
				}
			}
		})
	}
}


func TestClient_DeleteCard_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
		wantErrSub string
	}{
		{"Not Found", http.StatusNotFound, "Not Found", "error", "Not Found"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "error", "Server Error"},
		{"Network Error", 0, "", "error", "Delete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.statusCode == 0 {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				}))
				ts.Close()
			} else {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer ts.Close()
			}

			cfg := &config.Config{
				NCURL:    ts.URL,
				Timeout:  1 * time.Second,
			}
			cli := deck.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.DeleteCard(ctx, 1, 2, 3)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
			if tt.wantErrSub != "" && res["status"] == "error" && tt.statusCode != 0 {
				errStr, _ := res["error"].(string)
				if !strings.Contains(errStr, tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrSub, errStr)
				}
			}
		})
	}
}


func TestClient_newRequest_BadURL(t *testing.T) {
	cfg := &config.Config{
		NCURL:    "://invalid-url",
		Timeout:  1 * time.Second,
	}
	cli := deck.NewClient(cfg)
	ctx := context.Background()

	lRes, _ := cli.ListBoards(ctx)
	if lRes["status"] != "error" {
		t.Errorf("expected ListBoards to fail with bad URL")
	}
	sRes, _ := cli.ListStacks(ctx, 1)
	if sRes["status"] != "error" {
		t.Errorf("expected ListStacks to fail with bad URL")
	}
	cRes, _ := cli.CreateCard(ctx, 1, 2, "Title", "Desc")
	if cRes["status"] != "error" {
		t.Errorf("expected CreateCard to fail with bad URL")
	}
	uRes, _ := cli.UpdateCard(ctx, 1, 2, 3, "Title", "Desc", 0)
	if uRes["status"] != "error" {
		t.Errorf("expected UpdateCard to fail with bad URL")
	}
	dRes, _ := cli.DeleteCard(ctx, 1, 2, 3)
	if dRes["status"] != "error" {
		t.Errorf("expected DeleteCard to fail with bad URL")
	}
}

func TestClient_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Config{
		NCURL:    ts.URL,
		Timeout:  1 * time.Second,
	}
	cli := deck.NewClient(cfg)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lRes, _ := cli.ListBoards(ctx)
	if lRes["status"] != "error" {
		t.Errorf("expected ListBoards to fail on context cancel")
	}
	sRes, _ := cli.ListStacks(ctx, 1)
	if sRes["status"] != "error" {
		t.Errorf("expected ListStacks to fail on context cancel")
	}
	cRes, _ := cli.CreateCard(ctx, 1, 2, "Title", "Desc")
	if cRes["status"] != "error" {
		t.Errorf("expected CreateCard to fail on context cancel")
	}
	uRes, _ := cli.UpdateCard(ctx, 1, 2, 3, "Title", "Desc", 0)
	if uRes["status"] != "error" {
		t.Errorf("expected UpdateCard to fail on context cancel")
	}
	dRes, _ := cli.DeleteCard(ctx, 1, 2, 3)
	if dRes["status"] != "error" {
		t.Errorf("expected DeleteCard to fail on context cancel")
	}
}
