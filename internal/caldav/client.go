package caldav

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
)

// Client interacts with Nextcloud CalDAV API.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new CalDAV client.
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

// ListEvents lists events from Nextcloud CalDAV calendar via REPORT query.
func (c *Client) ListEvents(ctx context.Context, calendarName string) (map[string]any, error) {
	if calendarName == "" {
		calendarName = "personal"
	}
	targetURL := fmt.Sprintf("%s/", c.cfg.CalDAVURL(calendarName))

	reportXML := `<?xml version="1.0" encoding="utf-8" ?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop><d:getetag/><c:calendar-data/></d:prop>
  <c:filter><c:comp-filter name="VCALENDAR"/></c:filter>
</c:calendar-query>`

	req, err := c.newRequest(ctx, "REPORT", targetURL, strings.NewReader(reportXML))
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
		rawXML := string(respBytes)
		if len(rawXML) > 2000 {
			rawXML = rawXML[:2000]
		}
		return map[string]any{
			"status":         "success",
			"raw_caldav_xml": rawXML,
		}, nil
	}

	errText := string(respBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// CreateEvent creates a new event in Nextcloud CalDAV calendar using iCalendar (.ics).
func (c *Client) CreateEvent(ctx context.Context, eventUID, summary, dtstart, dtend, calendarName string) (map[string]any, error) {
	if calendarName == "" {
		calendarName = "personal"
	}
	targetURL := fmt.Sprintf("%s/%s.ics", c.cfg.CalDAVURL(calendarName), eventUID)

	icsContent := fmt.Sprintf(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//TheNovaNodes//Kairos Calendar//EN
BEGIN:VEVENT
UID:%s
SUMMARY:%s
DTSTART:%s
DTEND:%s
END:VEVENT
END:VCALENDAR`, eventUID, summary, dtstart, dtend)

	req, err := c.newRequest(ctx, http.MethodPut, targetURL, bytes.NewReader([]byte(icsContent)))
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		return map[string]any{
			"status":    "success",
			"event_uid": eventUID,
			"message":   "Event created",
		}, nil
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// DeleteEvent deletes an event from Nextcloud CalDAV calendar.
func (c *Client) DeleteEvent(ctx context.Context, eventUID, calendarName string) (map[string]any, error) {
	if calendarName == "" {
		calendarName = "personal"
	}
	targetURL := fmt.Sprintf("%s/%s.ics", c.cfg.CalDAVURL(calendarName), eventUID)

	req, err := c.newRequest(ctx, http.MethodDelete, targetURL, nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return map[string]any{
			"status":  "success",
			"message": "Event deleted",
		}, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return map[string]any{
			"status": "error",
			"error":  "Event not found",
		}, nil
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}
