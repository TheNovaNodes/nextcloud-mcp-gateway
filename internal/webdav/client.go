package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
)

// FileItem represents a single file or directory returned by WebDAV PROPFIND.
type FileItem struct {
	Href         string `json:"href"`
	IsDirectory  bool   `json:"is_directory"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
	ContentType  string `json:"content_type"`
}

// XML structures for WebDAV Multistatus response
type propfindResponse struct {
	XMLName  xml.Name `xml:"response"`
	Href     string   `xml:"href"`
	Propstat []struct {
		Prop struct {
			ResourceType struct {
				Collection *struct{} `xml:"collection"`
			} `xml:"resourcetype"`
			GetContentLength string `xml:"getcontentlength"`
			GetLastModified  string `xml:"getlastmodified"`
			GetContentType   string `xml:"getcontenttype"`
		} `xml:"prop"`
	} `xml:"propstat"`
}

type propfindMultistatus struct {
	XMLName   xml.Name           `xml:"multistatus"`
	Responses []propfindResponse `xml:"response"`
}

// Client interacts with Nextcloud WebDAV API.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new WebDAV client with timeout and connection pooling.
func NewClient(cfg *config.Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
	}
}

func (c *Client) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("User-Agent", "TheNovaNodes-Nextcloud-MCP-Gateway/1.0")
	req.Header.Set("Accept", "application/json, text/xml, */*")
	if c.cfg.Username != "" && c.cfg.Password != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
	return req, nil
}

// ListFiles queries a directory via PROPFIND Depth: 1 with pagination.
func (c *Client) ListFiles(ctx context.Context, rawPath string, offset, limit int) (map[string]any, error) {
	cleanP, err := NormalizePath(rawPath)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	targetURL := fmt.Sprintf("%s%s", c.cfg.WebDAVURL(), cleanP)
	if cleanP != "/" && !strings.HasSuffix(targetURL, "/") {
		// WebDAV servers expect a trailing slash for directories in PROPFIND
		targetURL += "/"
	}

	propfindXML := `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns" xmlns:nc="http://nextcloud.org/ns">
  <d:prop>
    <d:getlastmodified/>
    <d:getcontentlength/>
    <d:getcontenttype/>
    <d:resourcetype/>
    <oc:fileid/>
    <oc:size/>
  </d:prop>
</d:propfind>`

	req, err := c.newRequest(ctx, "PROPFIND", targetURL, strings.NewReader(propfindXML))
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer func() {
		_, _ = io.CopyN(io.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return map[string]any{"status": "error", "error": fmt.Sprintf("Path not found: %s", cleanP)}, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return map[string]any{"status": "error", "error": "Authentication failed. Provide valid NC_USER and NC_APP_PASSWORD."}, nil
	}
	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		limitErr := string(respBytes)
		if len(limitErr) > 300 {
			limitErr = limitErr[:300]
		}
		return map[string]any{"status": "error", "code": resp.StatusCode, "error": limitErr}, nil
	}

	var multi propfindMultistatus
	if err := xml.Unmarshal(respBytes, &multi); err != nil {
		rawXML := string(respBytes)
		if len(rawXML) > 1000 {
			rawXML = rawXML[:1000]
		}
		return map[string]any{
			"status":      "success",
			"raw_xml":     rawXML,
			"parse_error": err.Error(),
		}, nil
	}

	var items []FileItem
	for _, r := range multi.Responses {
		item := FileItem{Href: r.Href}
		for _, ps := range r.Propstat {
			isDir := ps.Prop.ResourceType.Collection != nil
			item.IsDirectory = isDir
			if isDir {
				item.ContentType = "directory"
			} else {
				item.ContentType = ps.Prop.GetContentType
				if item.ContentType == "" {
					item.ContentType = "file"
				}
			}
			if sz, err := strconv.ParseInt(ps.Prop.GetContentLength, 10, 64); err == nil {
				item.SizeBytes = sz
			}
			item.LastModified = ps.Prop.GetLastModified
		}
		items = append(items, item)
	}

	if limit <= 0 {
		limit = 50
	}
	safeLimit := limit
	if safeLimit > 100 {
		safeLimit = 100
	}
	if offset < 0 {
		offset = 0
	}

	totalCount := len(items)
	var paginated []FileItem
	if offset < totalCount {
		end := offset + safeLimit
		if end > totalCount {
			end = totalCount
		}
		paginated = items[offset:end]
	} else {
		paginated = []FileItem{}
	}

	return map[string]any{
		"status":         "success",
		"path":           cleanP,
		"total_count":    totalCount,
		"returned_count": len(paginated),
		"offset":         offset,
		"limit":          safeLimit,
		"items":          paginated,
	}, nil
}

// ReadFile retrieves the text content of a file.
func (c *Client) ReadFile(ctx context.Context, rawPath string) (map[string]any, error) {
	cleanP, err := NormalizePath(rawPath)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	targetURL := fmt.Sprintf("%s%s", c.cfg.WebDAVURL(), cleanP)
	req, err := c.newRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer func() {
		_, _ = io.CopyN(io.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	const limit = 10 << 20
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	if len(bodyBytes) > limit {
		return map[string]any{"status": "error", "error": fmt.Sprintf("file exceeds 10MB limit: %s", cleanP)}, nil
	}

	if resp.StatusCode == http.StatusOK {
		return map[string]any{
			"status":     "success",
			"path":       cleanP,
			"size_bytes": len(bodyBytes),
			"content":    string(bodyBytes),
		}, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return map[string]any{"status": "error", "error": fmt.Sprintf("File not found: %s", cleanP)}, nil
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// WriteFile creates or overwrites a file via WebDAV PUT.
func (c *Client) WriteFile(ctx context.Context, rawPath, content string) (map[string]any, error) {
	cleanP, err := NormalizePath(rawPath)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	targetURL := fmt.Sprintf("%s%s", c.cfg.WebDAVURL(), cleanP)
	data := []byte(content)
	req, err := c.newRequest(ctx, http.MethodPut, targetURL, bytes.NewReader(data))
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer func() {
		_, _ = io.CopyN(io.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 301))

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		return map[string]any{
			"status":        "success",
			"path":          cleanP,
			"bytes_written": len(data),
			"message":       "File written successfully",
		}, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return map[string]any{"status": "error", "error": "Unauthorized to write file in Nextcloud."}, nil
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// DeleteResource deletes a file or folder via WebDAV DELETE.
func (c *Client) DeleteResource(ctx context.Context, rawPath string) (map[string]any, error) {
	cleanP, err := NormalizePath(rawPath)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	targetURL := fmt.Sprintf("%s%s", c.cfg.WebDAVURL(), cleanP)
	req, err := c.newRequest(ctx, http.MethodDelete, targetURL, nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer func() {
		_, _ = io.CopyN(io.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 301))

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return map[string]any{"status": "success", "path": cleanP, "message": "Resource deleted"}, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return map[string]any{"status": "error", "error": fmt.Sprintf("Resource not found: %s", cleanP)}, nil
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// CreateFolder creates a new folder via WebDAV MKCOL.
func (c *Client) CreateFolder(ctx context.Context, rawPath string) (map[string]any, error) {
	cleanP, err := NormalizePath(rawPath)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	targetURL := fmt.Sprintf("%s%s", c.cfg.WebDAVURL(), cleanP)
	req, err := c.newRequest(ctx, "MKCOL", targetURL, nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer func() {
		_, _ = io.CopyN(io.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 301))

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return map[string]any{"status": "success", "path": cleanP, "message": "Folder created successfully"}, nil
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return map[string]any{"status": "exists", "path": cleanP, "message": "Folder already exists"}, nil
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}
