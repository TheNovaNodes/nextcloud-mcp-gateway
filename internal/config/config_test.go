package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
)

func TestLoadFromEnv_Defaults(t *testing.T) {
	os.Clearenv()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.NCURL != "http://127.0.0.1:8080" {
		t.Errorf("expected NCURL http://127.0.0.1:8080, got %s", cfg.NCURL)
	}
	if cfg.PublicURL != "https://nextcloud.example.com" {
		t.Errorf("expected PublicURL https://nextcloud.example.com, got %s", cfg.PublicURL)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.WebDAVURL() != "http://127.0.0.1:8080/remote.php/webdav" {
		t.Errorf("expected default WebDAV URL, got %s", cfg.WebDAVURL())
	}
	if cfg.CalDAVURL("") != "http://127.0.0.1:8080/remote.php/dav/calendars/current/personal" {
		t.Errorf("expected default CalDAV URL, got %s", cfg.CalDAVURL(""))
	}
}

func TestLoadFromEnv_Custom(t *testing.T) {
	os.Setenv("NC_URL", "https://cloud.example.com/")
	os.Setenv("NC_PUBLIC_URL", "https://cloud.example.com/")
	os.Setenv("NC_USER", "admin")
	os.Setenv("NC_APP_PASSWORD", "secret123")
	os.Setenv("NC_TIMEOUT", "15.5")
	defer os.Clearenv()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.NCURL != "https://cloud.example.com" {
		t.Errorf("expected trimmed NCURL, got %s", cfg.NCURL)
	}
	if cfg.Username != "admin" || cfg.Password != "secret123" {
		t.Errorf("expected admin credentials, got %s:%s", cfg.Username, cfg.Password)
	}
	if cfg.Timeout != 15500*time.Millisecond {
		t.Errorf("expected Timeout 15.5s, got %v", cfg.Timeout)
	}
	if cfg.WebDAVURL() != "https://cloud.example.com/remote.php/dav/files/admin" {
		t.Errorf("expected user WebDAV URL, got %s", cfg.WebDAVURL())
	}
	if cfg.CalDAVURL("work") != "https://cloud.example.com/remote.php/dav/calendars/admin/work" {
		t.Errorf("expected custom CalDAV URL, got %s", cfg.CalDAVURL("work"))
	}
	if cfg.DeckURL() != "https://cloud.example.com/index.php/apps/deck/api/v1.0" {
		t.Errorf("expected Deck URL, got %s", cfg.DeckURL())
	}
	if cfg.OCSURL() != "https://cloud.example.com/ocs/v1.php/cloud" {
		t.Errorf("expected OCS URL, got %s", cfg.OCSURL())
	}
	if cfg.StatusURL() != "https://cloud.example.com/status.php" {
		t.Errorf("expected Status URL, got %s", cfg.StatusURL())
	}
}
