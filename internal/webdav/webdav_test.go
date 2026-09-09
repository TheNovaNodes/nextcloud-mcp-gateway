package webdav_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"
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

func TestClient_ListFiles_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		path       string
		wantStatus string
		wantErrSub string
	}{
		{"Bad Path", http.StatusOK, "", "../../etc/passwd", "error", "traversal detected"},
		{"Unauthorized", http.StatusUnauthorized, "Auth required", "/docs", "error", "Authentication failed"},
		{"Forbidden", http.StatusForbidden, "Forbidden", "/docs", "error", "Authentication failed"},
		{"Not Found", http.StatusNotFound, "Not Found", "/docs", "error", "Path not found"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "/docs", "error", "Server Error"},
		{"Invalid XML", http.StatusMultiStatus, "<invalid>", "/docs", "success", ""}, // it actually returns success with raw_xml
		{"Network Error", 0, "", "/docs", "error", "Get"}, // Request error
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
			cli := webdav.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.ListFiles(ctx, tt.path, 0, 50)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if res["status"] != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res["status"])
			}
			
			if tt.wantErrSub != "" && res["status"] == "error" {
				errStr, _ := res["error"].(string)
				if !strings.Contains(errStr, tt.wantErrSub) && tt.statusCode != 0 {
					t.Errorf("expected error containing %q, got %q", tt.wantErrSub, errStr)
				}
			}
			
			if tt.name == "Invalid XML" {
			    if res["parse_error"] == nil {
			        t.Errorf("expected parse_error for Invalid XML")
			    }
			}
		})
	}
}

func TestClient_ReadFile_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		path       string
		wantStatus string
		wantErrSub string
	}{
		{"Bad Path", http.StatusOK, "", "../../etc/passwd", "error", "traversal detected"},
		{"Unauthorized", http.StatusUnauthorized, "Auth required", "/file.txt", "error", "Auth required"},
		{"Not Found", http.StatusNotFound, "Not Found", "/file.txt", "error", "File not found"},
		{"Network Error", 0, "", "/file.txt", "error", "Get"},
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
			cli := webdav.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.ReadFile(ctx, tt.path)
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


func TestClient_WriteFile_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		path       string
		wantStatus string
		wantErrSub string
	}{
		{"Bad Path", http.StatusOK, "", "../../etc/passwd", "error", "traversal detected"},
		{"Unauthorized", http.StatusUnauthorized, "Auth required", "/file.txt", "error", "Unauthorized to write file"},
		{"Forbidden", http.StatusForbidden, "Forbidden", "/file.txt", "error", "Unauthorized to write file"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "/file.txt", "error", "Server Error"},
		{"Network Error", 0, "", "/file.txt", "error", "Put"},
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
			cli := webdav.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.WriteFile(ctx, tt.path, "data")
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


func TestClient_DeleteResource_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		path       string
		wantStatus string
		wantErrSub string
	}{
		{"Bad Path", http.StatusOK, "", "../../etc/passwd", "error", "traversal detected"},
		{"Not Found", http.StatusNotFound, "Not Found", "/file.txt", "error", "Resource not found"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "/file.txt", "error", "Server Error"},
		{"Network Error", 0, "", "/file.txt", "error", "Delete"},
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
			cli := webdav.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.DeleteResource(ctx, tt.path)
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


func TestClient_CreateFolder_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		path       string
		wantStatus string
		wantErrSub string
	}{
		{"Bad Path", http.StatusOK, "", "../../etc/passwd", "error", "traversal detected"},
		{"Internal Server Error", http.StatusInternalServerError, "Server Error", "/folder", "error", "Server Error"},
		{"Network Error", 0, "", "/folder", "error", "MKCOL"},
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
			cli := webdav.NewClient(cfg)
			ctx := context.Background()

			res, err := cli.CreateFolder(ctx, tt.path)
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
	cli := webdav.NewClient(cfg)
	ctx := context.Background()

	lRes, _ := cli.ListFiles(ctx, "/", 0, 50)
	if lRes["status"] != "error" {
		t.Errorf("expected ListFiles to fail with bad URL")
	}
	rRes, _ := cli.ReadFile(ctx, "/f")
	if rRes["status"] != "error" {
		t.Errorf("expected ReadFile to fail with bad URL")
	}
	wRes, _ := cli.WriteFile(ctx, "/f", "d")
	if wRes["status"] != "error" {
		t.Errorf("expected WriteFile to fail with bad URL")
	}
	dRes, _ := cli.DeleteResource(ctx, "/f")
	if dRes["status"] != "error" {
		t.Errorf("expected DeleteResource to fail with bad URL")
	}
	cRes, _ := cli.CreateFolder(ctx, "/f")
	if cRes["status"] != "error" {
		t.Errorf("expected CreateFolder to fail with bad URL")
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
	cli := webdav.NewClient(cfg)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lRes, _ := cli.ListFiles(ctx, "/", 0, 50)
	if lRes["status"] != "error" {
		t.Errorf("expected ListFiles to fail on context cancel")
	}
	rRes, _ := cli.ReadFile(ctx, "/f")
	if rRes["status"] != "error" {
		t.Errorf("expected ReadFile to fail on context cancel")
	}
	wRes, _ := cli.WriteFile(ctx, "/f", "d")
	if wRes["status"] != "error" {
		t.Errorf("expected WriteFile to fail on context cancel")
	}
	dRes, _ := cli.DeleteResource(ctx, "/f")
	if dRes["status"] != "error" {
		t.Errorf("expected DeleteResource to fail on context cancel")
	}
	cRes, _ := cli.CreateFolder(ctx, "/f")
	if cRes["status"] != "error" {
		t.Errorf("expected CreateFolder to fail on context cancel")
	}
}
