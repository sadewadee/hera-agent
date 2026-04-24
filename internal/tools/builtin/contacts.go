package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sadewadee/hera/internal/tools"
)

// ContactsTool provides contact management capabilities.
type ContactsTool struct {
	mu       sync.Mutex
	contacts map[string]*Contact
	nextID   int
}

// Contact represents a contact entry.
type Contact struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
	Notes string `json:"notes,omitempty"`
}

type contactsArgs struct {
	Action string `json:"action"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Notes  string `json:"notes,omitempty"`
	Query  string `json:"query,omitempty"`
}

func (t *ContactsTool) Name() string { return "contacts" }

func (t *ContactsTool) Description() string {
	return "Manages contacts: create, list, search, update, and delete contacts."
}

func (t *ContactsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["create", "list", "search", "get", "update", "delete"],
				"description": "Contact action to perform."
			},
			"id": {"type": "string", "description": "Contact ID."},
			"name": {"type": "string", "description": "Contact name."},
			"email": {"type": "string", "description": "Contact email."},
			"phone": {"type": "string", "description": "Contact phone."},
			"notes": {"type": "string", "description": "Contact notes."},
			"query": {"type": "string", "description": "Search query (for search action)."}
		},
		"required": ["action"]
	}`)
}

func (t *ContactsTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a contactsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.contacts == nil {
		t.contacts = make(map[string]*Contact)
	}

	switch a.Action {
	case "create":
		if a.Name == "" {
			return &tools.Result{Content: "name is required", IsError: true}, nil
		}
		t.nextID++
		id := fmt.Sprintf("contact-%d", t.nextID)
		c := &Contact{ID: id, Name: a.Name, Email: a.Email, Phone: a.Phone, Notes: a.Notes}
		t.contacts[id] = c
		out, _ := json.Marshal(c)
		return &tools.Result{Content: string(out)}, nil

	case "list":
		list := make([]*Contact, 0, len(t.contacts))
		for _, c := range t.contacts {
			list = append(list, c)
		}
		out, _ := json.Marshal(list)
		return &tools.Result{Content: string(out)}, nil

	case "search":
		q := strings.ToLower(a.Query)
		var matches []*Contact
		for _, c := range t.contacts {
			if strings.Contains(strings.ToLower(c.Name), q) ||
				strings.Contains(strings.ToLower(c.Email), q) ||
				strings.Contains(strings.ToLower(c.Phone), q) {
				matches = append(matches, c)
			}
		}
		out, _ := json.Marshal(matches)
		return &tools.Result{Content: string(out)}, nil

	case "get":
		c, ok := t.contacts[a.ID]
		if !ok {
			return &tools.Result{Content: "contact not found", IsError: true}, nil
		}
		out, _ := json.Marshal(c)
		return &tools.Result{Content: string(out)}, nil

	case "update":
		c, ok := t.contacts[a.ID]
		if !ok {
			return &tools.Result{Content: "contact not found", IsError: true}, nil
		}
		if a.Name != "" {
			c.Name = a.Name
		}
		if a.Email != "" {
			c.Email = a.Email
		}
		if a.Phone != "" {
			c.Phone = a.Phone
		}
		if a.Notes != "" {
			c.Notes = a.Notes
		}
		out, _ := json.Marshal(c)
		return &tools.Result{Content: string(out)}, nil

	case "delete":
		if _, ok := t.contacts[a.ID]; !ok {
			return &tools.Result{Content: "contact not found", IsError: true}, nil
		}
		delete(t.contacts, a.ID)
		return &tools.Result{Content: fmt.Sprintf("Contact %s deleted", a.ID)}, nil

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

// RegisterContacts registers the contacts tool with the given registry.
func RegisterContacts(registry *tools.Registry) {
	registry.Register(&ContactsTool{})
}
