package mcp

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Permission represents a pending or resolved permission request.
type Permission struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`      // "command_exec", "file_write", "network_access"
	Resource   string     `json:"resource"`  // what's being accessed
	Requester  string     `json:"requester"` // who requested it
	Status     string     `json:"status"`    // "pending", "approved", "denied"
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// PermissionStore manages permission requests and responses.
// It is safe for concurrent use.
type PermissionStore struct {
	mu          sync.Mutex
	permissions map[string]*Permission
}

// NewPermissionStore creates an empty permission store.
func NewPermissionStore() *PermissionStore {
	return &PermissionStore{
		permissions: make(map[string]*Permission),
	}
}

// Request creates a new pending permission request and returns it.
func (ps *PermissionStore) Request(permType, resource, requester string) *Permission {
	p := &Permission{
		ID:        uuid.New().String(),
		Type:      permType,
		Resource:  resource,
		Requester: requester,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	ps.mu.Lock()
	ps.permissions[p.ID] = p
	ps.mu.Unlock()

	return p
}

// ListPending returns all permissions with status "pending".
func (ps *PermissionStore) ListPending() []*Permission {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	var result []*Permission
	for _, p := range ps.permissions {
		if p.Status == "pending" {
			result = append(result, p)
		}
	}
	return result
}

// Respond approves or denies a pending permission.
// action must be "approve" or "deny".
func (ps *PermissionStore) Respond(id, action string) (*Permission, error) {
	if action != "approve" && action != "deny" {
		return nil, fmt.Errorf("invalid action %q: must be 'approve' or 'deny'", action)
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	p, ok := ps.permissions[id]
	if !ok {
		return nil, fmt.Errorf("permission not found: %s", id)
	}

	if p.Status != "pending" {
		return nil, fmt.Errorf("permission %s is already %s", id, p.Status)
	}

	now := time.Now()
	p.ResolvedAt = &now

	if action == "approve" {
		p.Status = "approved"
	} else {
		p.Status = "denied"
	}

	return p, nil
}

// Get returns a permission by ID.
func (ps *PermissionStore) Get(id string) (*Permission, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	p, ok := ps.permissions[id]
	return p, ok
}
