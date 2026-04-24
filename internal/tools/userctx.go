package tools

import "context"

// userIDKey is the unexported context key used by WithUserID /
// UserIDFromContext so no unrelated code can collide on the string.
type userIDKey struct{}

// WithUserID returns a derived context that carries the given user ID.
// Used by the agent at tool dispatch time so per-user tools (memory,
// preferences) can fall back to this identity when the LLM's tool
// arguments omit the user_id field.
func WithUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromContext extracts the user ID attached by WithUserID. Returns
// an empty string if none was set — tools should then fall back to
// their own default (usually "default").
func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(userIDKey{}).(string); ok {
		return v
	}
	return ""
}
