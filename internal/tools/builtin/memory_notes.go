package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
)

// Typed memory notes — the "harness-style" auto-memory tools. These
// complement memory_save/memory_search (simple key/value facts) with
// a richer schema: four semantic types (user, feedback, project,
// reference), an addressable Name, a one-line Description shown in
// listings, and a freeform Content body.
//
// Together they let the agent:
//   - save durable knowledge about the user and ongoing work
//   - refine or remove a previous entry instead of duplicating
//   - list what's already remembered before considering a new save

// nameRE validates note names. Keep tight so the name is file-path-safe
// and copy-pasteable: lowercase letters, digits, dash, underscore, 1-64 chars.
var nameRE = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

// resolveNoteUserID picks the user ID for a note operation, preferring
// an explicit args.user_id, falling back to the session user ID the
// agent injected via context, and finally to the "default" bucket when
// nothing upstream knows who is talking. The agent wires
// tools.WithUserID(ctx, <session-user-id>) at dispatch time (see
// internal/agent/agent.go) so every memory write scopes to the actual
// platform user instead of pooling everyone under "default".
func resolveNoteUserID(ctx context.Context, argsUserID string) string {
	if id := strings.TrimSpace(argsUserID); id != "" {
		return id
	}
	if id := tools.UserIDFromContext(ctx); id != "" {
		return id
	}
	return "default"
}

// MemoryNoteSaveTool creates or overwrites a typed memory note.
type MemoryNoteSaveTool struct {
	manager *memory.Manager
}

type memoryNoteSaveArgs struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	UserID      string `json:"user_id,omitempty"`
}

func (t *MemoryNoteSaveTool) Name() string { return "memory_note_save" }

func (t *MemoryNoteSaveTool) Description() string {
	return `Saves a typed memory note that persists across conversations.

Use for durable information that will help in future sessions. Types:
- user: who the user is (role, goals, knowledge, preferences)
- feedback: guidance the user gave about how to work (corrections or validated approaches, include the why)
- project: ongoing work, goals, incidents, deadlines (use absolute dates, not relative)
- reference: pointers to external systems (Slack channels, dashboards, issue trackers)

Do NOT save: code patterns, paths, git history, debugging solutions, anything in CLAUDE.md, or ephemeral task state.

Call memory_note_list first to avoid duplicating existing notes — update instead.`
}

func (t *MemoryNoteSaveTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"type": {"type": "string", "enum": ["user", "feedback", "project", "reference"], "description": "Category of the note."},
			"name": {"type": "string", "description": "Short unique slug (lowercase, digits, dashes/underscores, <= 64 chars). Used to address the note later."},
			"description": {"type": "string", "description": "One-line hook shown in listings. Specific, under ~150 chars."},
			"content": {"type": "string", "description": "Full body. For feedback/project include a Why: and How to apply: line."},
			"user_id": {"type": "string", "description": "User identifier. Defaults to 'default'."}
		},
		"required": ["type", "name", "description", "content"]
	}`)
}

func (t *MemoryNoteSaveTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var p memoryNoteSaveArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	typ := memory.NoteType(strings.ToLower(strings.TrimSpace(p.Type)))
	if !memory.ValidNoteType(typ) {
		return &tools.Result{Content: "type must be one of: user, feedback, project, reference", IsError: true}, nil
	}
	name := strings.TrimSpace(p.Name)
	if !nameRE.MatchString(name) {
		return &tools.Result{Content: "name must be lowercase alphanumeric plus dash/underscore, 1-64 chars", IsError: true}, nil
	}
	desc := strings.TrimSpace(p.Description)
	content := strings.TrimSpace(p.Content)
	if desc == "" || content == "" {
		return &tools.Result{Content: "description and content are required", IsError: true}, nil
	}

	note := memory.Note{
		UserID:      resolveNoteUserID(ctx, p.UserID),
		Type:        typ,
		Name:        name,
		Description: desc,
		Content:     content,
	}
	if err := t.manager.SaveNote(ctx, note); err != nil {
		return &tools.Result{Content: fmt.Sprintf("save note: %v", err), IsError: true}, nil
	}
	return &tools.Result{Content: fmt.Sprintf("saved %s note %q", typ, name)}, nil
}

// MemoryNoteUpdateTool updates an existing note's description or content.
type MemoryNoteUpdateTool struct {
	manager *memory.Manager
}

type memoryNoteUpdateArgs struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

func (t *MemoryNoteUpdateTool) Name() string { return "memory_note_update" }

func (t *MemoryNoteUpdateTool) Description() string {
	return "Updates an existing memory note's description or content. Supply only the fields you want to change; omitted fields stay as-is."
}

func (t *MemoryNoteUpdateTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Slug of the note to update."},
			"description": {"type": "string", "description": "New one-line description. Omit to keep existing."},
			"content": {"type": "string", "description": "New body. Omit to keep existing."},
			"user_id": {"type": "string"}
		},
		"required": ["name"]
	}`)
}

func (t *MemoryNoteUpdateTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var p memoryNoteUpdateArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return &tools.Result{Content: "name is required", IsError: true}, nil
	}
	if strings.TrimSpace(p.Description) == "" && strings.TrimSpace(p.Content) == "" {
		return &tools.Result{Content: "at least one of description or content must be supplied", IsError: true}, nil
	}
	err := t.manager.UpdateNote(ctx, resolveNoteUserID(ctx, p.UserID), name,
		strings.TrimSpace(p.Description), strings.TrimSpace(p.Content))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("update note: %v", err), IsError: true}, nil
	}
	return &tools.Result{Content: fmt.Sprintf("updated note %q", name)}, nil
}

// MemoryNoteDeleteTool removes a note by name.
type MemoryNoteDeleteTool struct {
	manager *memory.Manager
}

type memoryNoteDeleteArgs struct {
	Name   string `json:"name"`
	UserID string `json:"user_id,omitempty"`
}

func (t *MemoryNoteDeleteTool) Name() string { return "memory_note_delete" }

func (t *MemoryNoteDeleteTool) Description() string {
	return "Removes a memory note permanently. Use when a note is outdated or the user asks to forget something."
}

func (t *MemoryNoteDeleteTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Slug of the note to delete."},
			"user_id": {"type": "string"}
		},
		"required": ["name"]
	}`)
}

func (t *MemoryNoteDeleteTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var p memoryNoteDeleteArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return &tools.Result{Content: "name is required", IsError: true}, nil
	}
	if err := t.manager.DeleteNote(ctx, resolveNoteUserID(ctx, p.UserID), name); err != nil {
		return &tools.Result{Content: fmt.Sprintf("delete note: %v", err), IsError: true}, nil
	}
	return &tools.Result{Content: fmt.Sprintf("deleted note %q", name)}, nil
}

// MemoryNoteListTool returns an index of the user's notes.
type MemoryNoteListTool struct {
	manager *memory.Manager
}

type memoryNoteListArgs struct {
	Type   string `json:"type,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

func (t *MemoryNoteListTool) Name() string { return "memory_note_list" }

func (t *MemoryNoteListTool) Description() string {
	return "Lists memory notes for the user. Optional type filter (user, feedback, project, reference). Returns name + description per entry. Use this before saving to avoid duplicates."
}

func (t *MemoryNoteListTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"type": {"type": "string", "enum": ["user", "feedback", "project", "reference"], "description": "Optional type filter."},
			"user_id": {"type": "string"}
		}
	}`)
}

func (t *MemoryNoteListTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var p memoryNoteListArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	var typ memory.NoteType
	if p.Type != "" {
		typ = memory.NoteType(strings.ToLower(strings.TrimSpace(p.Type)))
		if !memory.ValidNoteType(typ) {
			return &tools.Result{Content: "type must be one of: user, feedback, project, reference", IsError: true}, nil
		}
	}
	notes, err := t.manager.ListNotes(ctx, resolveNoteUserID(ctx, p.UserID), typ)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("list notes: %v", err), IsError: true}, nil
	}
	if len(notes) == 0 {
		return &tools.Result{Content: "(no notes saved)"}, nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d note(s):\n", len(notes)))
	for _, n := range notes {
		sb.WriteString(fmt.Sprintf("- [%s] %s — %s\n", n.Type, n.Name, n.Description))
	}
	return &tools.Result{Content: sb.String()}, nil
}

// MemoryNoteGetTool returns the full body of a single note.
type MemoryNoteGetTool struct {
	manager *memory.Manager
}

type memoryNoteGetArgs struct {
	Name   string `json:"name"`
	UserID string `json:"user_id,omitempty"`
}

func (t *MemoryNoteGetTool) Name() string { return "memory_note_get" }

func (t *MemoryNoteGetTool) Description() string {
	return "Returns the full body of a memory note by name."
}

func (t *MemoryNoteGetTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Slug of the note."},
			"user_id": {"type": "string"}
		},
		"required": ["name"]
	}`)
}

func (t *MemoryNoteGetTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var p memoryNoteGetArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return &tools.Result{Content: "name is required", IsError: true}, nil
	}
	n, err := t.manager.GetNote(ctx, resolveNoteUserID(ctx, p.UserID), name)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("get note: %v", err), IsError: true}, nil
	}
	if n == nil {
		return &tools.Result{Content: fmt.Sprintf("no note named %q", name), IsError: true}, nil
	}
	return &tools.Result{Content: fmt.Sprintf("[%s] %s\n%s\n\n%s", n.Type, n.Name, n.Description, n.Content)}, nil
}

// RegisterMemoryNotes registers the 5 note-management tools.
func RegisterMemoryNotes(registry *tools.Registry, mgr *memory.Manager) {
	registry.Register(&MemoryNoteSaveTool{manager: mgr})
	registry.Register(&MemoryNoteUpdateTool{manager: mgr})
	registry.Register(&MemoryNoteDeleteTool{manager: mgr})
	registry.Register(&MemoryNoteListTool{manager: mgr})
	registry.Register(&MemoryNoteGetTool{manager: mgr})
}
