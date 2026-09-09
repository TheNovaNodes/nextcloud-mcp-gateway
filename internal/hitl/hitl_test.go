package hitl_test

import (
	"testing"
	"time"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/hitl"
)

func TestHITLManager_RequestAndPop(t *testing.T) {
	mgr := hitl.NewManager(10, 5*time.Minute)

	details := map[string]any{"path": "/Documents/test.txt", "content": "hello"}
	res := mgr.Request("write_file", details)

	if res["status"] != "pending_approval" {
		t.Fatalf("expected pending_approval, got %v", res["status"])
	}

	token, ok := res["token"].(string)
	if !ok || token == "" {
		t.Fatalf("expected valid string token, got %v", res["token"])
	}

	if mgr.Len() != 1 {
		t.Errorf("expected 1 pending action, got %d", mgr.Len())
	}

	act, err := mgr.Pop(token)
	if err != nil {
		t.Fatalf("failed to pop action: %v", err)
	}

	if act.Type != "write_file" {
		t.Errorf("expected write_file, got %s", act.Type)
	}
	if act.Details["path"] != "/Documents/test.txt" {
		t.Errorf("expected path /Documents/test.txt, got %v", act.Details["path"])
	}

	// Double pop must fail
	_, err = mgr.Pop(token)
	if err == nil {
		t.Errorf("expected error on double pop, got nil")
	}
}

func TestHITLManager_Expiration(t *testing.T) {
	mgr := hitl.NewManager(10, 20*time.Millisecond)

	res := mgr.Request("delete_file", map[string]any{"path": "/test"})
	token := res["token"].(string)

	time.Sleep(30 * time.Millisecond)

	_, err := mgr.Pop(token)
	if err == nil {
		t.Fatalf("expected expired token error, got nil")
	}
}

func TestHITLManager_CapacityEviction(t *testing.T) {
	mgr := hitl.NewManager(2, 10*time.Second)

	res1 := mgr.Request("action1", nil)
	tok1 := res1["token"].(string)

	time.Sleep(2 * time.Millisecond)
	res2 := mgr.Request("action2", nil)
	tok2 := res2["token"].(string)

	time.Sleep(2 * time.Millisecond)
	res3 := mgr.Request("action3", nil)
	tok3 := res3["token"].(string)

	// tok1 must have been evicted as oldest
	_, err := mgr.Pop(tok1)
	if err == nil {
		t.Errorf("expected tok1 to be evicted, but popped successfully")
	}

	// tok2 and tok3 must still exist
	if _, err := mgr.Pop(tok2); err != nil {
		t.Errorf("expected tok2 to exist: %v", err)
	}
	if _, err := mgr.Pop(tok3); err != nil {
		t.Errorf("expected tok3 to exist: %v", err)
	}
}
