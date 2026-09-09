package webdav_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/webdav"
)

func TestClient_ListFiles(t *testing.T) {
	xmlResp := `<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/remote.php/dav/files/admin/Documents/</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype><d:collection/></d:resourcetype>
        <d:getlastmodified>Wed, 09 Sep 2026 12:00:00 GMT</d:getlastmodified>
        <d:getcontentlength>0</d:getcontentlength>
      </d:prop>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/remote.php/dav/files/admin/Documents/report.pdf</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype/>
        <d:getcontentlength>12345</d:getcontentlength>
        <d:getlastmodified>Wed, 09 Sep 2026 12:30:00 GMT</d:getlastmodified>
        <d:getcontenttype>application/pdf</d:getcontenttype>
      </d:prop>
    </d:propstat>
  </d:response>
</d:multistatus>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Errorf("expected PROPFIND, got %s", r.Method)
		}
		if r.Header.Get("Depth") != "1" {
			t.Errorf("expected Depth: 1, got %s", r.Header.Get("Depth"))
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer ts.Close()

	cfg := &config.Config{
		NCURL:    ts.URL,
		Username: "admin",
		Password: "password",
		Timeout:  5 * time.Second,
	}
	cli := webdav.NewClient(cfg)

	res, err := cli.ListFiles(context.Background(), "/Documents", 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res["status"] != "success" {
		t.Fatalf("expected status success, got %v", res["status"])
	}
	if res["total_count"] != 2 {
		t.Errorf("expected total_count 2, got %v", res["total_count"])
	}
	items := res["items"].([]webdav.FileItem)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if !items[0].IsDirectory {
		t.Errorf("expected item 0 to be directory")
	}
	if items[1].SizeBytes != 12345 {
		t.Errorf("expected item 1 size 12345, got %d", items[1].SizeBytes)
	}
	if items[1].ContentType != "application/pdf" {
		t.Errorf("expected content-type application/pdf, got %s", items[1].ContentType)
	}
}

func TestClient_ReadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/remote.php/dav/files/admin/note.txt" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Kairos test content"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cfg := &config.Config{
		NCURL:    ts.URL,
		Username: "admin",
		Password: "password",
		Timeout:  5 * time.Second,
	}
	cli := webdav.NewClient(cfg)

	// Found
	res, err := cli.ReadFile(context.Background(), "/note.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "success" || res["content"] != "Kairos test content" {
		t.Errorf("unexpected read result: %v", res)
	}

	// Not found
	resNotFound, err := cli.ReadFile(context.Background(), "/missing.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resNotFound["status"] != "error" {
		t.Errorf("expected status error for missing file, got %v", resNotFound)
	}
}

func TestClient_WriteDeleteFolder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case "MKCOL":
			if r.URL.Path == "/remote.php/dav/files/admin/existing" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusCreated)
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
	cli := webdav.NewClient(cfg)
	ctx := context.Background()

	// Write
	wRes, err := cli.WriteFile(ctx, "/file.txt", "content")
	if err != nil || wRes["status"] != "success" {
		t.Errorf("unexpected write result: %v, err: %v", wRes, err)
	}

	// Delete
	dRes, err := cli.DeleteResource(ctx, "/file.txt")
	if err != nil || dRes["status"] != "success" {
		t.Errorf("unexpected delete result: %v, err: %v", dRes, err)
	}

	// MKCOL new
	mRes, err := cli.CreateFolder(ctx, "/new_folder")
	if err != nil || mRes["status"] != "success" {
		t.Errorf("unexpected mkcol result: %v, err: %v", mRes, err)
	}

	// MKCOL existing
	mResEx, err := cli.CreateFolder(ctx, "/existing")
	if err != nil || mResEx["status"] != "exists" {
		t.Errorf("unexpected mkcol existing result: %v, err: %v", mResEx, err)
	}
}
