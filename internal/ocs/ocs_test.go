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
