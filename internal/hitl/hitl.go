package hitl

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// DefaultMaxPending is the maximum capacity of pending actions to prevent DoS/memory leak.
	DefaultMaxPending = 100
	// DefaultTTL is 5 minutes (300 seconds), matching security requirements.
	DefaultTTL = 300 * time.Second
)

// Action represents a staged mutating action awaiting human approval.
type Action struct {
	Type      string         `json:"type"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"created_at"`
}

// Manager coordinates human-in-the-loop validation and token staging.
type Manager struct {
	mu         sync.Mutex
	pending    map[string]*Action
	maxPending int
	ttl        time.Duration
}

// NewManager creates a new HITL Manager instance.
func NewManager(maxPending int, ttl time.Duration) *Manager {
	if maxPending <= 0 {
		maxPending = DefaultMaxPending
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Manager{
		pending:    make(map[string]*Action),
		maxPending: maxPending,
		ttl:        ttl,
	}
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CleanupExpired removes actions older than TTL.
func (m *Manager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cleanupExpiredLocked()
}

func (m *Manager) cleanupExpiredLocked() int {
	now := time.Now()
	cleaned := 0
	for tok, act := range m.pending {
		if now.Sub(act.CreatedAt) > m.ttl {
			delete(m.pending, tok)
			cleaned++
		}
	}
	return cleaned
}

// Request stages a mutating action, generates a token, and returns the response payload.
func (m *Manager) Request(actionType string, details map[string]any) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanupExpiredLocked()

	// Evict oldest action if capacity reached
	if len(m.pending) >= m.maxPending {
		var oldestTok string
		var oldestTime time.Time
		first := true
		for tok, act := range m.pending {
			if first || act.CreatedAt.Before(oldestTime) {
				oldestTok = tok
				oldestTime = act.CreatedAt
				first = false
			}
		}
		if oldestTok != "" {
			delete(m.pending, oldestTok)
		}
	}

	token := generateToken()
	m.pending[token] = &Action{
		Type:      actionType,
		Details:   details,
		CreatedAt: time.Now(),
	}

	return map[string]any{
		"status":  "pending_approval",
		"message": fmt.Sprintf("🚨 HITL REQUIRED: Action '%s' is blocked. To confirm, use 'execute_pending_action' with token.", actionType),
		"token":   token,
	}
}

// Pop atomically retrieves and deletes a staged action for execution.
func (m *Manager) Pop(token string) (*Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanupExpiredLocked()

	act, ok := m.pending[token]
	if !ok {
		return nil, errors.New("Invalid, expired, or already executed HITL token.")
	}

	if time.Since(act.CreatedAt) > m.ttl {
		delete(m.pending, token)
		return nil, errors.New("HITL token expired.")
	}

	delete(m.pending, token)
	return act, nil
}

// Len returns the current count of pending actions.
func (m *Manager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.pending)
}
