package environments

import "context"

// ToolContext provides context data to tools during environment execution.
type ToolContext struct {
	ctx         context.Context
	environment string
	sessionID   string
	workDir     string
	metadata    map[string]interface{}
}

// NewToolContext creates a new tool context.
func NewToolContext(ctx context.Context, env, sessionID, workDir string) *ToolContext {
	return &ToolContext{ctx: ctx, environment: env, sessionID: sessionID, workDir: workDir, metadata: make(map[string]interface{})}
}

// Context returns the underlying Go context.
func (tc *ToolContext) Context() context.Context { return tc.ctx }

// Environment returns the environment name.
func (tc *ToolContext) Environment() string { return tc.environment }

// SessionID returns the session ID.
func (tc *ToolContext) SessionID() string { return tc.sessionID }

// WorkDir returns the working directory.
func (tc *ToolContext) WorkDir() string { return tc.workDir }

// Set stores a metadata value.
func (tc *ToolContext) Set(key string, value interface{}) { tc.metadata[key] = value }

// Get retrieves a metadata value.
func (tc *ToolContext) Get(key string) (interface{}, bool) { v, ok := tc.metadata[key]; return v, ok }
