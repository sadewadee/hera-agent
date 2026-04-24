package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
)

// The session tools let the agent browse past conversation sessions
// and pull one back into the active turn. They complement the
// always-on typed-note layer: notes carry durable identity, these
// tools carry the actual dialogue history on demand.

// SessionListTool returns metadata for the current user's recent
// sessions. No message bodies — just IDs, timestamps, counts, and
// the opening user message as a preview. Follow up with
// session_recall when the user picks one.
type SessionListTool struct {
	manager *memory.Manager
}

type sessionListArgs struct {
	Limit  int    `json:"limit,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

func (t *SessionListTool) Name() string { return "session_list" }

func (t *SessionListTool) Description() string {
	return `Returns the user's recent conversation sessions with IDs, message counts, timestamps, and a short preview of the opening message.

Call when the user asks to browse, resume, or summarize past sessions ("tunjukkan sesi sebelumnya", "percakapan kemarin", "what did we talk about last time"). The result lists sessions; follow up with session_recall <session_id> once the user picks one.

Do NOT invent session IDs — only use IDs returned from this tool.`
}

func (t *SessionListTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"limit": {"type": "integer", "description": "Max sessions to return. Defaults to 5, capped at 20."},
			"user_id": {"type": "string", "description": "User identifier. Defaults to current session user."}
		}
	}`)
}

func (t *SessionListTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var p sessionListArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	userID := resolveNoteUserID(ctx, p.UserID)
	limit := p.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	sessions, err := t.manager.ListUserSessions(ctx, userID, limit)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("list sessions: %v", err), IsError: true}, nil
	}
	if len(sessions) == 0 {
		return &tools.Result{Content: "(no past sessions found for this user)"}, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d past session(s) for user %q:\n", len(sessions), userID)
	for i, s := range sessions {
		last := "unknown"
		if !s.LastMessage.IsZero() {
			last = humanTimeAgo(s.LastMessage)
		}
		preview := s.Preview
		if preview == "" {
			preview = "(no user message)"
		}
		fmt.Fprintf(&sb, "  %d. id=%s last=%s msgs=%d preview=%q\n",
			i+1, s.SessionID, last, s.MessageCount, preview)
	}
	sb.WriteString("\nUse session_recall with one of these ids to pull its transcript / summary into this turn.")
	return &tools.Result{Content: sb.String()}, nil
}

// SessionRecallTool loads a past session and returns either its raw
// transcript or an LLM-generated summary, so the agent can answer
// "what did we talk about in session X" without the user having to
// paste history.
type SessionRecallTool struct {
	manager *memory.Manager
}

type sessionRecallArgs struct {
	SessionID string `json:"session_id"`
	Summarize bool   `json:"summarize,omitempty"`
	MaxChars  int    `json:"max_chars,omitempty"`
	UserID    string `json:"user_id,omitempty"`
}

func (t *SessionRecallTool) Name() string { return "session_recall" }

func (t *SessionRecallTool) Description() string {
	return `Loads a past conversation session by ID and returns either a condensed summary (default) or the raw transcript.

Only call with session_id values that came from session_list in this same conversation — do not invent IDs.

Set summarize=false to get the raw transcript (capped by max_chars, default 8000) when the user explicitly wants to see the full messages.

Sessions belonging to a different user are refused; only the active user's sessions are accessible.`
}

func (t *SessionRecallTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"session_id": {"type": "string", "description": "Session ID exactly as returned by session_list."},
			"summarize":  {"type": "boolean", "description": "true (default) returns an LLM-generated summary; false returns the raw transcript capped by max_chars."},
			"max_chars":  {"type": "integer", "description": "Cap on raw transcript length when summarize=false. Default 8000."},
			"user_id":    {"type": "string", "description": "User identifier. Defaults to current session user."}
		},
		"required": ["session_id"]
	}`)
}

func (t *SessionRecallTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var p sessionRecallArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	sid := strings.TrimSpace(p.SessionID)
	if sid == "" {
		return &tools.Result{Content: "session_id is required", IsError: true}, nil
	}

	// Scope enforcement: the session ID format is "platform:userID:uuid".
	// Require the current user to be embedded in the ID before touching
	// the session so user A can't fetch user B's history by guessing.
	userID := resolveNoteUserID(ctx, p.UserID)
	if !strings.Contains(sid, ":"+userID+":") {
		return &tools.Result{
			Content: fmt.Sprintf("session %q does not belong to user %q", sid, userID),
			IsError: true,
		}, nil
	}

	messages, err := t.manager.GetConversation(ctx, sid)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("load session: %v", err), IsError: true}, nil
	}
	if len(messages) == 0 {
		return &tools.Result{Content: fmt.Sprintf("session %q has no messages", sid)}, nil
	}

	summarize := p.Summarize
	// Default: summarize. Falsey zero-value of bool works here because
	// the user flag is the primary way to opt into raw transcript.
	// If caller wants explicit default, we flip here.
	if !argsHasSummarizeKey(args) {
		summarize = true
	}

	if summarize {
		summary, err := t.manager.SummarizeSession(ctx, sid)
		if err != nil {
			// Fall back to head+tail transcript so recall still
			// works when the summarizer (LLM) is unavailable.
			return &tools.Result{Content: formatTranscript(sid, messages, 4000) +
				fmt.Sprintf("\n\n(note: summary unavailable: %v)", err)}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("Session %s — summary:\n\n%s", sid, summary.Content)}, nil
	}

	maxChars := p.MaxChars
	if maxChars <= 0 {
		maxChars = 8000
	}
	return &tools.Result{Content: formatTranscript(sid, messages, maxChars)}, nil
}

// argsHasSummarizeKey tells us whether the caller explicitly set
// summarize or not, so the default-true behaviour only kicks in when
// the key is absent from JSON (vs explicitly false).
func argsHasSummarizeKey(raw json.RawMessage) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m["summarize"]
	return ok
}

func formatTranscript(sid string, messages []llm.Message, maxChars int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Session %s — %d messages\n\n", sid, len(messages))
	for _, m := range messages {
		line := fmt.Sprintf("[%s] %s\n", m.Role, m.Content)
		if sb.Len()+len(line) > maxChars {
			sb.WriteString("... (transcript truncated)\n")
			break
		}
		sb.WriteString(line)
	}
	return sb.String()
}

// humanTimeAgo renders a rough "2h ago" style string so session
// listings stay compact. Not intended for precise timestamps.
func humanTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// RegisterSessionTools registers session_list and session_recall.
// Only wired when a memory manager is available (same precondition
// as the other memory tools).
func RegisterSessionTools(registry *tools.Registry, mgr *memory.Manager) {
	registry.Register(&SessionListTool{manager: mgr})
	registry.Register(&SessionRecallTool{manager: mgr})
}
