// Package builtin provides built-in tool implementations.
//
// session_search.go implements the session search tool that provides
// long-term conversation recall by searching past session transcripts
// via SQLite FTS5 and summarising matching sessions.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// SessionSearchTool searches past conversation sessions.
type SessionSearchTool struct {
	db SessionDB
}

// SessionDB defines the interface for session database operations needed
// by the session search tool. Implementations are provided by the state
// package.
type SessionDB interface {
	SearchMessages(query string, roleFilter []string, excludeSources []string, limit, offset int) ([]SessionMatch, error)
	GetSession(sessionID string) (*SessionMeta, error)
	GetSessionMessages(sessionID string) ([]SessionMessage, error)
	ListSessionsRich(limit int, excludeSources []string) ([]SessionListEntry, error)
}

// SessionMatch represents a search result from FTS5.
type SessionMatch struct {
	SessionID      string `json:"session_id"`
	Source         string `json:"source"`
	Model          string `json:"model,omitempty"`
	SessionStarted int64  `json:"session_started,omitempty"`
}

// SessionMeta holds session metadata.
type SessionMeta struct {
	ID              string `json:"id"`
	Source          string `json:"source"`
	ParentSessionID string `json:"parent_session_id,omitempty"`
	StartedAt       int64  `json:"started_at,omitempty"`
}

// SessionMessage represents a conversation message.
type SessionMessage struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	ToolName string `json:"tool_name,omitempty"`
}

// SessionListEntry represents a session in a recent-sessions listing.
type SessionListEntry struct {
	ID           string `json:"session_id"`
	Title        string `json:"title,omitempty"`
	Source       string `json:"source"`
	StartedAt    string `json:"started_at"`
	LastActive   string `json:"last_active"`
	MessageCount int    `json:"message_count"`
	Preview      string `json:"preview,omitempty"`
}

// HiddenSessionSources are excluded from session browsing by default.
var HiddenSessionSources = []string{"tool"}

const (
	maxSessionChars  = 100_000
	maxSummaryTokens = 10000
)

// RegisterSessionSearch registers the session search tool.
func RegisterSessionSearch(registry *tools.Registry, db SessionDB) {
	if db == nil {
		return
	}
	tool := &SessionSearchTool{db: db}
	registry.Register(tool)
}

func (t *SessionSearchTool) Name() string { return "session_search" }
func (t *SessionSearchTool) Description() string {
	return "Search past conversation sessions or browse recent sessions for long-term recall"
}

func (t *SessionSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query keywords for past sessions. Omit to browse recent sessions."
			},
			"role_filter": {
				"type": "string",
				"description": "Comma-separated roles to filter (e.g. user,assistant)"
			},
			"limit": {
				"type": "integer",
				"description": "Max sessions to return (default 3, max 5)"
			}
		}
	}`)
}

func (t *SessionSearchTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params struct {
		Query      string `json:"query"`
		RoleFilter string `json:"role_filter"`
		Limit      int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 3
	}
	if limit > 5 {
		limit = 5
	}

	query := strings.TrimSpace(params.Query)

	// Recent sessions mode: no query, return metadata only.
	if query == "" {
		return t.listRecentSessions(limit)
	}

	return t.searchSessions(ctx, query, params.RoleFilter, limit)
}

func (t *SessionSearchTool) listRecentSessions(limit int) (*tools.Result, error) {
	sessions, err := t.db.ListSessionsRich(limit+5, HiddenSessionSources)
	if err != nil {
		slog.Error("failed to list recent sessions", "error", err)
		return &tools.Result{Content: fmt.Sprintf("failed to list recent sessions: %v", err), IsError: true}, nil
	}

	// Filter out child/delegation sessions.
	var results []SessionListEntry
	for _, s := range sessions {
		if len(results) >= limit {
			break
		}
		results = append(results, s)
	}

	resp := map[string]any{
		"success": true,
		"mode":    "recent",
		"results": results,
		"count":   len(results),
		"message": fmt.Sprintf("Showing %d most recent sessions.", len(results)),
	}
	data, _ := json.Marshal(resp)
	return &tools.Result{Content: string(data)}, nil
}

func (t *SessionSearchTool) searchSessions(ctx context.Context, query, roleFilter string, limit int) (*tools.Result, error) {
	// Parse role filter.
	var roleList []string
	if roleFilter != "" {
		for _, r := range strings.Split(roleFilter, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				roleList = append(roleList, r)
			}
		}
	}

	// FTS5 search.
	matches, err := t.db.SearchMessages(query, roleList, HiddenSessionSources, 50, 0)
	if err != nil {
		slog.Error("session search failed", "error", err)
		return &tools.Result{Content: fmt.Sprintf("search failed: %v", err), IsError: true}, nil
	}

	if len(matches) == 0 {
		resp := map[string]any{
			"success": true,
			"query":   query,
			"results": []any{},
			"count":   0,
			"message": "No matching sessions found.",
		}
		data, _ := json.Marshal(resp)
		return &tools.Result{Content: string(data)}, nil
	}

	// Deduplicate by session ID.
	seen := make(map[string]SessionMatch)
	for _, m := range matches {
		if _, ok := seen[m.SessionID]; !ok {
			seen[m.SessionID] = m
		}
		if len(seen) >= limit {
			break
		}
	}

	// Build summaries from matched sessions.
	var summaries []map[string]any
	for sessionID, matchInfo := range seen {
		messages, err := t.db.GetSessionMessages(sessionID)
		if err != nil || len(messages) == 0 {
			continue
		}

		conversationText := formatConversation(messages)
		conversationText = truncateAroundMatches(conversationText, query, maxSessionChars)

		entry := map[string]any{
			"session_id": sessionID,
			"when":       formatTimestamp(matchInfo.SessionStarted),
			"source":     matchInfo.Source,
		}
		if matchInfo.Model != "" {
			entry["model"] = matchInfo.Model
		}

		// Use raw preview (summarization requires auxiliary LLM not available here).
		preview := conversationText
		if len(preview) > 500 {
			preview = preview[:500] + "\n...[truncated]"
		}
		entry["summary"] = preview
		summaries = append(summaries, entry)
	}

	resp := map[string]any{
		"success":           true,
		"query":             query,
		"results":           summaries,
		"count":             len(summaries),
		"sessions_searched": len(seen),
	}
	data, _ := json.Marshal(resp)
	return &tools.Result{Content: string(data)}, nil
}

// formatConversation formats session messages into a readable transcript.
func formatConversation(messages []SessionMessage) string {
	var parts []string
	for _, msg := range messages {
		role := strings.ToUpper(msg.Role)
		content := msg.Content
		if role == "TOOL" && msg.ToolName != "" {
			if len(content) > 500 {
				content = content[:250] + "\n...[truncated]...\n" + content[len(content)-250:]
			}
			parts = append(parts, fmt.Sprintf("[TOOL:%s]: %s", msg.ToolName, content))
		} else {
			parts = append(parts, fmt.Sprintf("[%s]: %s", role, content))
		}
	}
	return strings.Join(parts, "\n\n")
}

// truncateAroundMatches truncates a conversation transcript to maxChars,
// centered around where query terms appear.
func truncateAroundMatches(fullText, query string, maxChars int) string {
	if len(fullText) <= maxChars {
		return fullText
	}

	queryTerms := strings.Fields(strings.ToLower(query))
	textLower := strings.ToLower(fullText)
	firstMatch := len(fullText)
	for _, term := range queryTerms {
		pos := strings.Index(textLower, term)
		if pos != -1 && pos < firstMatch {
			firstMatch = pos
		}
	}
	if firstMatch == len(fullText) {
		firstMatch = 0
	}

	half := maxChars / 2
	start := firstMatch - half
	if start < 0 {
		start = 0
	}
	end := start + maxChars
	if end > len(fullText) {
		end = len(fullText)
		start = end - maxChars
		if start < 0 {
			start = 0
		}
	}

	truncated := fullText[start:end]
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "...[earlier conversation truncated]...\n\n"
	}
	if end < len(fullText) {
		suffix = "\n\n...[later conversation truncated]..."
	}
	return prefix + truncated + suffix
}

// formatTimestamp converts a Unix timestamp to a human-readable date.
func formatTimestamp(ts int64) string {
	if ts == 0 {
		return "unknown"
	}
	t := time.Unix(ts, 0)
	return t.Format("January 2, 2006 at 3:04 PM")
}
