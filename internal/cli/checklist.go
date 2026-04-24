package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ChecklistItem represents a single todo item.
type ChecklistItem struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// Checklist manages a list of todo items with persistence.
type Checklist struct {
	mu       sync.Mutex
	items    []ChecklistItem
	nextID   int
	filePath string
}

// NewChecklist creates or loads a checklist from the given file path.
// If filePath is empty, it defaults to ~/.hera/checklist.json.
func NewChecklist(filePath string) (*Checklist, error) {
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		dir := filepath.Join(home, ".hera")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create .hera dir: %w", err)
		}
		filePath = filepath.Join(dir, "checklist.json")
	}

	cl := &Checklist{
		items:    make([]ChecklistItem, 0),
		nextID:   1,
		filePath: filePath,
	}

	// Load existing items if the file exists.
	if data, err := os.ReadFile(filePath); err == nil {
		var items []ChecklistItem
		if json.Unmarshal(data, &items) == nil {
			cl.items = items
			for _, item := range items {
				if item.ID >= cl.nextID {
					cl.nextID = item.ID + 1
				}
			}
		}
	}

	return cl, nil
}

// Add adds a new item to the checklist and persists.
func (cl *Checklist) Add(text string) ChecklistItem {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	item := ChecklistItem{
		ID:        cl.nextID,
		Text:      text,
		Done:      false,
		CreatedAt: time.Now(),
	}
	cl.nextID++
	cl.items = append(cl.items, item)
	cl.saveLocked()
	return item
}

// Remove removes an item by ID.
func (cl *Checklist) Remove(id int) bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for i, item := range cl.items {
		if item.ID == id {
			cl.items = append(cl.items[:i], cl.items[i+1:]...)
			cl.saveLocked()
			return true
		}
	}
	return false
}

// Toggle flips the done status of an item by ID.
func (cl *Checklist) Toggle(id int) bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for i, item := range cl.items {
		if item.ID == id {
			cl.items[i].Done = !item.Done
			cl.saveLocked()
			return true
		}
	}
	return false
}

// List returns a formatted string of all checklist items.
func (cl *Checklist) List() string {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if len(cl.items) == 0 {
		return "Checklist is empty."
	}

	var sb strings.Builder
	sb.WriteString("Checklist:\n")
	for _, item := range cl.items {
		check := "[ ]"
		if item.Done {
			check = "[x]"
		}
		sb.WriteString(fmt.Sprintf("  %d. %s %s\n", item.ID, check, item.Text))
	}

	total := len(cl.items)
	done := 0
	for _, item := range cl.items {
		if item.Done {
			done++
		}
	}
	sb.WriteString(fmt.Sprintf("\n%d/%d completed", done, total))
	return sb.String()
}

// Items returns a copy of all items.
func (cl *Checklist) Items() []ChecklistItem {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cp := make([]ChecklistItem, len(cl.items))
	copy(cp, cl.items)
	return cp
}

// saveLocked persists the checklist to disk. Caller must hold the mutex.
func (cl *Checklist) saveLocked() {
	data, err := json.MarshalIndent(cl.items, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(cl.filePath, data, 0o644)
}
