package caldav_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/caldav"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
)

func TestClient_CalDAV(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "REPORT":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte("<d:multistatus><d:response>event</d:response></d:multistatus>"))
		case http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			if r.URL.Path == "/remote.php/dav/calendars/admin/personal/missing.ics" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	cfg := &config.Config{
		NCURL:    ts.URL,
		Username: "admin",
		Password: "password",
		Timeout:  5 * time.Second,
	}
	cli := caldav.NewClient(cfg)
	ctx := context.Background()

	// List
	listRes, err := cli.ListEvents(ctx, "personal")
	if err != nil || listRes["status"] != "success" {
		t.Fatalf("unexpected list result: %v, err: %v", listRes, err)
	}

	// Create
	createRes, err := cli.CreateEvent(ctx, "uid123", "Sync Meeting", "20260909T140000Z", "20260909T150000Z", "personal")
	if err != nil || createRes["status"] != "success" {
		t.Fatalf("unexpected create result: %v, err: %v", createRes, err)
	}

	// Delete existing
	delRes, err := cli.DeleteEvent(ctx, "uid123", "personal")
	if err != nil || delRes["status"] != "success" {
		t.Fatalf("unexpected delete result: %v, err: %v", delRes, err)
	}

	// Delete missing
	delMissing, err := cli.DeleteEvent(ctx, "missing", "personal")
	if err != nil || delMissing["status"] != "error" {
		t.Fatalf("unexpected delete missing result: %v, err: %v", delMissing, err)
	}
}
