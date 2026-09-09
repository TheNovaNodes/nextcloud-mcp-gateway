package ocs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
)

// Client interacts with Nextcloud OCS Cloud API and system status.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new OCS client.
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

// GetUserInfo retrieves storage quota, display name, and details via Nextcloud OCS API.
func (c *Client) GetUserInfo(ctx context.Context) (map[string]any, error) {
	user := c.cfg.Username
	if user == "" {
		user = "current"
	}
	targetURL := fmt.Sprintf("%s/users/%s?format=json", c.cfg.OCSURL(), user)

	req, err := c.newRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var ocsWrapper struct {
			OCS struct {
				Data struct {
					DisplayName     string         `json:"displayname"`
					Email           string         `json:"email"`
					Quota           map[string]any `json:"quota"`
					StorageLocation string         `json:"storageLocation"`
				} `json:"data"`
			} `json:"ocs"`
		}

		if err := json.Unmarshal(bodyBytes, &ocsWrapper); err == nil {
			return map[string]any{
				"status":           "success",
				"user":             user,
				"display_name":     ocsWrapper.OCS.Data.DisplayName,
				"email":            ocsWrapper.OCS.Data.Email,
				"quota":            ocsWrapper.OCS.Data.Quota,
				"storage_location": ocsWrapper.OCS.Data.StorageLocation,
			}, nil
		}
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// HealthCheck checks Nextcloud instance health, version, status.php.
func (c *Client) HealthCheck(ctx context.Context) (map[string]any, error) {
	statusURL := c.cfg.StatusURL()

	report := map[string]any{
		"status":        "unknown",
		"endpoint":      c.cfg.NCURL,
		"public_url":    c.cfg.PublicURL,
		"authenticated": c.cfg.Username != "" && c.cfg.Password != "",
		"user":          c.cfg.Username,
		"details":       map[string]any{},
	}
	if report["user"] == "" {
		report["user"] = "anonymous"
	}

	req, err := c.newRequest(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		report["status"] = "unreachable"
		report["error"] = err.Error()
		return report, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		report["status"] = "unreachable"
		report["error"] = err.Error()
		return report, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var data map[string]any
		if err := json.Unmarshal(bodyBytes, &data); err == nil {
			installed, _ := data["installed"].(bool)
			if installed {
				report["status"] = "healthy"
			} else {
				report["status"] = "maintenance"
			}
			report["details"] = data
		} else {
			report["status"] = "healthy"
			raw := string(bodyBytes)
			if len(raw) > 200 {
				raw = raw[:200]
			}
			report["details"] = map[string]any{"raw": raw}
		}
	} else {
		report["status"] = "degraded"
		raw := string(bodyBytes)
		if len(raw) > 200 {
			raw = raw[:200]
		}
		report["error"] = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, raw)
	}

	return report, nil
}
