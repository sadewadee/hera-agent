package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sadewadee/hera/internal/tools"
)

type TodoTool struct {
	mu    sync.Mutex
	items []todoItem
}

type todoItem struct {
	ID    int    `json:"id"`
	Text  string `json:"text"`
	Done  bool   `json:"done"`
}

type todoArgs struct {
	Action string `json:"action"`
	Text   string `json:"text,omitempty"`
	ID     int    `json:"id,omitempty"`
}

func (t *TodoTool) Name() string        { return "todo" }
func (t *TodoTool) Description() string  { return "Manages a TODO list: add, list, complete, and remove items." }
func (t *TodoTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["add","list","complete","remove"],"description":"Action to perform"},"text":{"type":"string","description":"Todo text (for add)"},"id":{"type":"integer","description":"Todo ID (for complete/remove)"}},"required":["action"]}`)
}

func (t *TodoTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a todoArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	switch a.Action {
	case "add":
		if a.Text == "" { return &tools.Result{Content: "text required", IsError: true}, nil }
		id := len(t.items) + 1
		t.items = append(t.items, todoItem{ID: id, Text: a.Text})
		return &tools.Result{Content: fmt.Sprintf("Added todo #%d: %s", id, a.Text)}, nil
	case "list":
		if len(t.items) == 0 { return &tools.Result{Content: "No todos."}, nil }
		var b strings.Builder
		for _, item := range t.items {
			mark := "[ ]"; if item.Done { mark = "[x]" }
			fmt.Fprintf(&b, "%s #%d: %s\n", mark, item.ID, item.Text)
		}
		return &tools.Result{Content: b.String()}, nil
	case "complete":
		for i, item := range t.items {
			if item.ID == a.ID { t.items[i].Done = true; return &tools.Result{Content: fmt.Sprintf("Completed #%d", a.ID)}, nil }
		}
		return &tools.Result{Content: "not found", IsError: true}, nil
	case "remove":
		for i, item := range t.items {
			if item.ID == a.ID { t.items = append(t.items[:i], t.items[i+1:]...); return &tools.Result{Content: fmt.Sprintf("Removed #%d", a.ID)}, nil }
		}
		return &tools.Result{Content: "not found", IsError: true}, nil
	default:
		return &tools.Result{Content: "unknown action", IsError: true}, nil
	}
}

func RegisterTodo(registry *tools.Registry) { registry.Register(&TodoTool{}) }
