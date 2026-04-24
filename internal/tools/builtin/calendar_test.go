package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCalendarTool_CreateAndList(t *testing.T) {
	tool := &CalendarTool{}
	ctx := context.Background()

	// Create an event
	args := `{"action":"create","title":"Meeting","start_time":"2026-04-12T10:00:00Z"}`
	result, err := tool.Execute(ctx, json.RawMessage(args))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if result.IsError {
		t.Fatalf("create returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "evt-1") {
		t.Errorf("expected event ID in response, got: %s", result.Content)
	}

	// List events
	listArgs := `{"action":"list"}`
	result, err = tool.Execute(ctx, json.RawMessage(listArgs))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(result.Content, "Meeting") {
		t.Errorf("list should contain event title, got: %s", result.Content)
	}
}

func TestCalendarTool_GetAndDelete(t *testing.T) {
	tool := &CalendarTool{}
	ctx := context.Background()

	// Create
	tool.Execute(ctx, json.RawMessage(`{"action":"create","title":"Test","start_time":"2026-04-12T09:00:00Z"}`))

	// Get
	result, _ := tool.Execute(ctx, json.RawMessage(`{"action":"get","id":"evt-1"}`))
	if result.IsError {
		t.Fatalf("get returned error: %s", result.Content)
	}

	// Delete
	result, _ = tool.Execute(ctx, json.RawMessage(`{"action":"delete","id":"evt-1"}`))
	if result.IsError {
		t.Fatalf("delete returned error: %s", result.Content)
	}

	// Get should now fail
	result, _ = tool.Execute(ctx, json.RawMessage(`{"action":"get","id":"evt-1"}`))
	if !result.IsError {
		t.Error("get after delete should return error")
	}
}

func TestCalendarTool_CreateMissingFields(t *testing.T) {
	tool := &CalendarTool{}
	ctx := context.Background()

	result, _ := tool.Execute(ctx, json.RawMessage(`{"action":"create"}`))
	if !result.IsError {
		t.Error("create without title/start_time should error")
	}
}

func TestCalendarTool_InvalidAction(t *testing.T) {
	tool := &CalendarTool{}
	ctx := context.Background()

	result, _ := tool.Execute(ctx, json.RawMessage(`{"action":"invalid"}`))
	if !result.IsError {
		t.Error("invalid action should return error")
	}
}
