package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// CalendarTool provides calendar event management capabilities.
type CalendarTool struct {
	mu     sync.Mutex
	events map[string]*CalendarEvent
	nextID int
}

// CalendarEvent represents a calendar entry.
type CalendarEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time,omitempty"`
	Location    string `json:"location,omitempty"`
}

type calendarArgs struct {
	Action      string `json:"action"`
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	StartTime   string `json:"start_time,omitempty"`
	EndTime     string `json:"end_time,omitempty"`
	Location    string `json:"location,omitempty"`
	Date        string `json:"date,omitempty"`
}

func (t *CalendarTool) Name() string { return "calendar" }

func (t *CalendarTool) Description() string {
	return "Manages calendar events: create, list, get, update, and delete events."
}

func (t *CalendarTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["create", "list", "get", "update", "delete"],
				"description": "Calendar action to perform."
			},
			"id": {
				"type": "string",
				"description": "Event ID (for get, update, delete)."
			},
			"title": {
				"type": "string",
				"description": "Event title."
			},
			"description": {
				"type": "string",
				"description": "Event description."
			},
			"start_time": {
				"type": "string",
				"description": "Start time in RFC3339 format."
			},
			"end_time": {
				"type": "string",
				"description": "End time in RFC3339 format."
			},
			"location": {
				"type": "string",
				"description": "Event location."
			},
			"date": {
				"type": "string",
				"description": "Date filter for list action (YYYY-MM-DD)."
			}
		},
		"required": ["action"]
	}`)
}

func (t *CalendarTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a calendarArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.events == nil {
		t.events = make(map[string]*CalendarEvent)
	}

	switch a.Action {
	case "create":
		if a.Title == "" || a.StartTime == "" {
			return &tools.Result{Content: "title and start_time are required", IsError: true}, nil
		}
		t.nextID++
		id := fmt.Sprintf("evt-%d", t.nextID)
		evt := &CalendarEvent{
			ID: id, Title: a.Title, Description: a.Description,
			StartTime: a.StartTime, EndTime: a.EndTime, Location: a.Location,
		}
		t.events[id] = evt
		out, _ := json.Marshal(evt)
		return &tools.Result{Content: string(out)}, nil

	case "list":
		evts := make([]*CalendarEvent, 0, len(t.events))
		for _, e := range t.events {
			if a.Date != "" {
				st, err := time.Parse(time.RFC3339, e.StartTime)
				if err != nil {
					continue
				}
				if st.Format("2006-01-02") != a.Date {
					continue
				}
			}
			evts = append(evts, e)
		}
		sort.Slice(evts, func(i, j int) bool { return evts[i].StartTime < evts[j].StartTime })
		out, _ := json.Marshal(evts)
		return &tools.Result{Content: string(out)}, nil

	case "get":
		evt, ok := t.events[a.ID]
		if !ok {
			return &tools.Result{Content: "event not found", IsError: true}, nil
		}
		out, _ := json.Marshal(evt)
		return &tools.Result{Content: string(out)}, nil

	case "update":
		evt, ok := t.events[a.ID]
		if !ok {
			return &tools.Result{Content: "event not found", IsError: true}, nil
		}
		if a.Title != "" {
			evt.Title = a.Title
		}
		if a.Description != "" {
			evt.Description = a.Description
		}
		if a.StartTime != "" {
			evt.StartTime = a.StartTime
		}
		if a.EndTime != "" {
			evt.EndTime = a.EndTime
		}
		if a.Location != "" {
			evt.Location = a.Location
		}
		out, _ := json.Marshal(evt)
		return &tools.Result{Content: string(out)}, nil

	case "delete":
		if _, ok := t.events[a.ID]; !ok {
			return &tools.Result{Content: "event not found", IsError: true}, nil
		}
		delete(t.events, a.ID)
		return &tools.Result{Content: fmt.Sprintf("Event %s deleted", a.ID)}, nil

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

// RegisterCalendar registers the calendar tool with the given registry.
func RegisterCalendar(registry *tools.Registry) {
	registry.Register(&CalendarTool{})
}
