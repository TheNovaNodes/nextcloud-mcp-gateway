package deck_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/deck"
)

func TestClient_Deck(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 1, "title": "Main Board"}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 10, "title": "Todo"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks/10/cards":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": 100, "title": "New Task"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks/10/cards/100":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "title": "Updated Task"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/index.php/apps/deck/api/v1.0/boards/1/stacks/10/cards/100":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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
	cli := deck.NewClient(cfg)
	ctx := context.Background()

	// List boards
	boardsRes, err := cli.ListBoards(ctx)
	if err != nil || boardsRes["status"] != "success" {
		t.Fatalf("unexpected boards result: %v, err: %v", boardsRes, err)
	}

	// List stacks
	stacksRes, err := cli.ListStacks(ctx, 1)
	if err != nil || stacksRes["status"] != "success" {
		t.Fatalf("unexpected stacks result: %v, err: %v", stacksRes, err)
	}

	// Create card
	createRes, err := cli.CreateCard(ctx, 1, 10, "New Task", "Desc")
	if err != nil || createRes["status"] != "success" {
		t.Fatalf("unexpected create card result: %v, err: %v", createRes, err)
	}

	// Update card
	updateRes, err := cli.UpdateCard(ctx, 1, 10, 100, "Updated Task", "Desc", 1)
	if err != nil || updateRes["status"] != "success" {
		t.Fatalf("unexpected update card result: %v, err: %v", updateRes, err)
	}

	// Delete card
	delRes, err := cli.DeleteCard(ctx, 1, 10, 100)
	if err != nil || delRes["status"] != "success" {
		t.Fatalf("unexpected delete card result: %v, err: %v", delRes, err)
	}
}
