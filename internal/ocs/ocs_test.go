package ocs_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/ocs"
)

func TestClient_OCS_And_Health(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status.php":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"installed": true, "version": "29.0.0"}`))
		case "/ocs/v1.php/cloud/users/admin":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ocs": {"data": {"displayname": "Administrator", "email": "admin@example.com", "quota": {"free": 1000}, "storageLocation": "/var/www"}}}`))
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
	cli := ocs.NewClient(cfg)
	ctx := context.Background()

	// Health
	hRes, err := cli.HealthCheck(ctx)
	if err != nil || hRes["status"] != "healthy" {
		t.Fatalf("unexpected health result: %v, err: %v", hRes, err)
	}

	// User Info
	uRes, err := cli.GetUserInfo(ctx)
	if err != nil || uRes["status"] != "success" {
		t.Fatalf("unexpected user info result: %v, err: %v", uRes, err)
	}
	if uRes["display_name"] != "Administrator" || uRes["email"] != "admin@example.com" {
		t.Errorf("unexpected user info data: %v", uRes)
	}
}

func TestClient_GetUserInfo_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
		wantErrSub string
	}{
		{"Unauthorized", http.StatusUnauthorized, "Auth required", "error", "Auth required"},
		{"Forbidden", http.StatusForbidden, "Forbidden", "error", "Forbidden"},
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
				ts.Close() // Force network error
			} else {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer ts.Close()
			}

			cfg := &config.Config{
				NCURL:    ts.URL,
				Username: "admin",
				Password: "password",
				Timeout:  1 * time.Second,
			}
			cli := ocs.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.GetUserInfo(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
			
			if tt.wantErrSub != "" {
				errStr, _ := res["error"].(string)
				if errStr == "" && tt.statusCode == 0 { // special case network error where res["error"] will contain Get "url"
                    if res["error"] == nil || len(res["error"].(string)) == 0 {
                        t.Errorf("expected error string containing %q, got none", tt.wantErrSub)
                    }
				}
			}
		})
	}
}

func TestClient_GetUserInfo_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Config{
		NCURL:    ts.URL,
		Timeout:  1 * time.Second,
	}
	cli := ocs.NewClient(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	res, err := cli.GetUserInfo(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "error" {
		t.Errorf("expected status error, got %v", res["status"])
	}
}


func TestClient_HealthCheck_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
	}{
		{"Unauthorized", http.StatusUnauthorized, "Auth required", "degraded"},
		{"Forbidden", http.StatusForbidden, "Forbidden", "degraded"},
		{"Not Found", http.StatusNotFound, "Not Found", "degraded"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "degraded"},
		{"Maintenance Mode", http.StatusOK, `{"installed": false}`, "maintenance"},
		{"Invalid JSON", http.StatusOK, "{invalid}", "healthy"},
		{"Network Error", 0, "", "unreachable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.statusCode == 0 {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				}))
				ts.Close() // Force network error
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
			cli := ocs.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.HealthCheck(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
		})
	}
}

func TestClient_HealthCheck_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Config{
		NCURL:    ts.URL,
		Timeout:  1 * time.Second,
	}
	cli := ocs.NewClient(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	res, err := cli.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "unreachable" {
		t.Errorf("expected status unreachable, got %v", res["status"])
	}
}

func TestClient_newRequest_BadURL(t *testing.T) {
	cfg := &config.Config{
		NCURL:    "://invalid-url",
		Timeout:  1 * time.Second,
	}
	cli := ocs.NewClient(cfg)
	ctx := context.Background()

	// This should fail during newRequest which GetUserInfo and HealthCheck call
	uRes, _ := cli.GetUserInfo(ctx)
	if uRes["status"] != "error" {
		t.Errorf("expected GetUserInfo to fail with bad URL")
	}

	hRes, _ := cli.HealthCheck(ctx)
	if hRes["status"] != "unreachable" {
		t.Errorf("expected HealthCheck to fail with bad URL")
	}
}
