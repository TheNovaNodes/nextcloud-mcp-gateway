package deck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
)

// Client interacts with Nextcloud Deck REST API v1.0.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new Deck API client.
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

// ListBoards retrieves all Kanban boards available to the user.
func (c *Client) ListBoards(ctx context.Context) (map[string]any, error) {
	targetURL := fmt.Sprintf("%s/boards", c.cfg.DeckURL())
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
		var boards any
		if err := json.Unmarshal(bodyBytes, &boards); err == nil {
			return map[string]any{
				"status": "success",
				"boards": boards,
			}, nil
		}
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// ListStacks retrieves all stacks (columns) in a Deck board.
func (c *Client) ListStacks(ctx context.Context, boardID int) (map[string]any, error) {
	targetURL := fmt.Sprintf("%s/boards/%d/stacks", c.cfg.DeckURL(), boardID)
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
		var stacks any
		if err := json.Unmarshal(bodyBytes, &stacks); err == nil {
			return map[string]any{
				"status": "success",
				"stacks": stacks,
			}, nil
		}
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// CreateCard creates a new Kanban card in a Deck stack.
func (c *Client) CreateCard(ctx context.Context, boardID, stackID int, title, description string) (map[string]any, error) {
	targetURL := fmt.Sprintf("%s/boards/%d/stacks/%d/cards", c.cfg.DeckURL(), boardID, stackID)
	payload := map[string]any{
		"title":       title,
		"description": description,
		"type":        "plain",
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := c.newRequest(ctx, http.MethodPost, targetURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		var card any
		if err := json.Unmarshal(bodyBytes, &card); err == nil {
			return map[string]any{
				"status": "success",
				"card":   card,
			}, nil
		}
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// UpdateCard updates an existing Kanban card (content, stack, or order).
func (c *Client) UpdateCard(ctx context.Context, boardID, stackID, cardID int, title, description string, order int) (map[string]any, error) {
	targetURL := fmt.Sprintf("%s/boards/%d/stacks/%d/cards/%d", c.cfg.DeckURL(), boardID, stackID, cardID)
	user := c.cfg.Username
	if user == "" {
		user = "admin"
	}
	payload := map[string]any{
		"title":       title,
		"description": description,
		"order":       order,
		"stackId":     stackID,
		"type":        "plain",
		"owner":       user,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := c.newRequest(ctx, http.MethodPut, targetURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var card any
		if err := json.Unmarshal(bodyBytes, &card); err == nil {
			return map[string]any{
				"status": "success",
				"card":   card,
			}, nil
		}
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}

// DeleteCard deletes a Kanban card in Nextcloud Deck.
func (c *Client) DeleteCard(ctx context.Context, boardID, stackID, cardID int) (map[string]any, error) {
	targetURL := fmt.Sprintf("%s/boards/%d/stacks/%d/cards/%d", c.cfg.DeckURL(), boardID, stackID, cardID)
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

	if resp.StatusCode == http.StatusOK {
		return map[string]any{
			"status":  "success",
			"message": "Card deleted",
		}, nil
	}

	errText := string(bodyBytes)
	if len(errText) > 300 {
		errText = errText[:300]
	}
	return map[string]any{"status": "error", "code": resp.StatusCode, "error": errText}, nil
}
