package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/memory"
)

// Deps holds dependencies for MCP tool handlers.
type Deps struct {
	Agent       *agent.Agent
	Memory      *memory.Manager
	Sessions    *agent.SessionManager
	EventBus    *EventBus
	Attachments *AttachmentStore
	Permissions *PermissionStore
}

// RegisterAllTools registers all 10 MCP tools with the server.
func RegisterAllTools(s *Server, deps Deps) {
	registerConversationsList(s, deps)
	registerConversationGet(s, deps)
	registerMessagesRead(s, deps)
	registerMessagesSend(s, deps)
	registerAttachmentsFetch(s, deps)
	registerEventsPoll(s, deps)
	registerEventsWait(s, deps)
	registerChannelsList(s, deps)
	registerPermissionsListOpen(s, deps)
	registerPermissionsRespond(s, deps)
}

func registerConversationsList(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "conversations_list",
		Description: "List all active conversation sessions.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		sessions := deps.Sessions.List()
		result := make([]map[string]any, 0, len(sessions))
		for _, sess := range sessions {
			result = append(result, map[string]any{
				"id":         sess.ID,
				"platform":   sess.Platform,
				"user_id":    sess.UserID,
				"turn_count": sess.TurnCount,
				"created_at": sess.CreatedAt,
				"updated_at": sess.UpdatedAt,
			})
		}
		return map[string]any{"conversations": result, "count": len(result)}, nil
	})
}

func registerConversationGet(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "conversation_get",
		Description: "Get details of a specific conversation session by ID.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string","description":"The session ID to look up."}},"required":["session_id"]}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		var args struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		sess, ok := deps.Sessions.Get(args.SessionID)
		if !ok {
			return nil, fmt.Errorf("session not found: %s", args.SessionID)
		}
		return map[string]any{
			"id":            sess.ID,
			"platform":      sess.Platform,
			"user_id":       sess.UserID,
			"turn_count":    sess.TurnCount,
			"message_count": len(sess.GetMessages()),
			"created_at":    sess.CreatedAt,
			"updated_at":    sess.UpdatedAt,
		}, nil
	})
}

func registerMessagesRead(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "messages_read",
		Description: "Read messages from a conversation session.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string","description":"Session ID to read from."},"limit":{"type":"integer","description":"Max messages to return. Defaults to 50."}},"required":["session_id"]}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		var args struct {
			SessionID string `json:"session_id"`
			Limit     int    `json:"limit"`
		}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if args.Limit <= 0 {
			args.Limit = 50
		}

		sess, ok := deps.Sessions.Get(args.SessionID)
		if !ok {
			// Try from persistent memory.
			if deps.Memory != nil {
				msgs, err := deps.Memory.GetConversation(ctx, args.SessionID)
				if err != nil {
					return nil, fmt.Errorf("session not found: %s", args.SessionID)
				}
				if len(msgs) > args.Limit {
					msgs = msgs[len(msgs)-args.Limit:]
				}
				result := make([]map[string]any, 0, len(msgs))
				for _, m := range msgs {
					result = append(result, map[string]any{
						"role":    string(m.Role),
						"content": m.Content,
					})
				}
				return map[string]any{"messages": result, "count": len(result)}, nil
			}
			return nil, fmt.Errorf("session not found: %s", args.SessionID)
		}

		msgs := sess.GetMessages()
		if len(msgs) > args.Limit {
			msgs = msgs[len(msgs)-args.Limit:]
		}
		result := make([]map[string]any, 0, len(msgs))
		for _, m := range msgs {
			result = append(result, map[string]any{
				"role":      string(m.Role),
				"content":   m.Content,
				"timestamp": m.Timestamp,
			})
		}
		return map[string]any{"messages": result, "count": len(result)}, nil
	})
}

func registerMessagesSend(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "messages_send",
		Description: "Send a message to the agent and get a response.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"Message text to send."},"platform":{"type":"string","description":"Platform name. Defaults to 'mcp'."},"user_id":{"type":"string","description":"User ID. Defaults to 'mcp-user'."}},"required":["text"]}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		var args struct {
			Text     string `json:"text"`
			Platform string `json:"platform"`
			UserID   string `json:"user_id"`
		}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if args.Platform == "" {
			args.Platform = "mcp"
		}
		if args.UserID == "" {
			args.UserID = "mcp-user"
		}
		if deps.Agent == nil {
			return nil, fmt.Errorf("agent not initialized")
		}
		response, err := deps.Agent.HandleMessage(ctx, args.Platform, "mcp-chat", args.UserID, args.Text)
		if err != nil {
			return nil, fmt.Errorf("agent error: %w", err)
		}
		return map[string]any{"response": response}, nil
	})
}

func registerAttachmentsFetch(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "attachments_fetch",
		Description: "Fetch attachment metadata for a session. Returns info about media attached to messages.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string","description":"Session ID to check for attachments."}},"required":["session_id"]}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		var args struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if deps.Attachments == nil {
			return map[string]any{"attachments": []any{}, "count": 0}, nil
		}
		atts := deps.Attachments.List(args.SessionID)
		result := make([]map[string]any, 0, len(atts))
		for _, a := range atts {
			result = append(result, map[string]any{
				"id":        a.ID,
				"type":      a.Type,
				"url":       a.URL,
				"name":      a.Name,
				"timestamp": a.Timestamp,
			})
		}
		return map[string]any{"attachments": result, "count": len(result)}, nil
	})
}

func registerEventsPoll(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "events_poll",
		Description: "Poll for new events (messages, state changes) since a given timestamp.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"since":{"type":"string","description":"ISO 8601 timestamp. Events after this time are returned."}}}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		// Return current session state as events.
		sessions := deps.Sessions.List()
		events := make([]map[string]any, 0)
		for _, sess := range sessions {
			events = append(events, map[string]any{
				"type":       "session_active",
				"session_id": sess.ID,
				"turn_count": sess.TurnCount,
				"updated_at": sess.UpdatedAt,
			})
		}
		return map[string]any{"events": events, "count": len(events)}, nil
	})
}

func registerEventsWait(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "events_wait",
		Description: "Wait for the next event (blocking). Returns when a new message or state change occurs.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"timeout_ms":{"type":"integer","description":"Maximum wait time in milliseconds. Defaults to 30000."}}}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		var args struct {
			TimeoutMs int `json:"timeout_ms"`
		}
		_ = json.Unmarshal(params, &args)
		if args.TimeoutMs <= 0 {
			args.TimeoutMs = 30000
		}

		// If no EventBus is wired, fall back to immediate timeout response.
		if deps.EventBus == nil {
			return map[string]any{"event": "timeout"}, nil
		}

		ch := deps.EventBus.Subscribe()
		defer deps.EventBus.Unsubscribe(ch)

		select {
		case evt := <-ch:
			return map[string]any{"event": evt.Type, "data": evt.Data}, nil
		case <-time.After(time.Duration(args.TimeoutMs) * time.Millisecond):
			return map[string]any{"event": "timeout"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
}

func registerChannelsList(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "channels_list",
		Description: "List available channels/platforms the agent is connected to.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		// Report sessions grouped by platform.
		sessions := deps.Sessions.List()
		platforms := make(map[string]int)
		for _, sess := range sessions {
			platforms[sess.Platform]++
		}
		channels := make([]map[string]any, 0, len(platforms))
		for plat, count := range platforms {
			channels = append(channels, map[string]any{
				"platform":        plat,
				"active_sessions": count,
			})
		}
		return map[string]any{"channels": channels}, nil
	})
}

func registerPermissionsListOpen(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "permissions_list_open",
		Description: "List pending permission requests (e.g., dangerous command approvals).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		if deps.Permissions == nil {
			return map[string]any{"permissions": []any{}, "count": 0}, nil
		}
		pending := deps.Permissions.ListPending()
		result := make([]map[string]any, 0, len(pending))
		for _, p := range pending {
			result = append(result, map[string]any{
				"id":         p.ID,
				"type":       p.Type,
				"resource":   p.Resource,
				"requester":  p.Requester,
				"status":     p.Status,
				"created_at": p.CreatedAt,
			})
		}
		return map[string]any{"permissions": result, "count": len(result)}, nil
	})
}

func registerPermissionsRespond(s *Server, deps Deps) {
	s.RegisterTool(ToolSchema{
		Name:        "permissions_respond",
		Description: "Respond to a pending permission request (approve or deny).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"permission_id":{"type":"string","description":"ID of the permission request."},"action":{"type":"string","enum":["approve","deny"],"description":"Whether to approve or deny."}},"required":["permission_id","action"]}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		var args struct {
			PermissionID string `json:"permission_id"`
			Action       string `json:"action"`
		}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if deps.Permissions == nil {
			return map[string]any{
				"status":        "acknowledged",
				"permission_id": args.PermissionID,
				"action":        args.Action,
			}, nil
		}
		perm, err := deps.Permissions.Respond(args.PermissionID, args.Action)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"status":        perm.Status,
			"permission_id": perm.ID,
			"action":        args.Action,
			"resolved_at":   perm.ResolvedAt,
		}, nil
	})
}
