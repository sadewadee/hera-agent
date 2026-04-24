package builtin_hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootMDHook_Name(t *testing.T) {
	t.Parallel()

	hook := NewBootMDHook("/nonexistent/boot.md")
	assert.Equal(t, "boot_md", hook.Name())
}

func TestBootMDHook_BeforeMessage_WithValidFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	bootPath := filepath.Join(tmpDir, "boot.md")
	err := os.WriteFile(bootPath, []byte("System instructions here."), 0644)
	require.NoError(t, err)

	hook := NewBootMDHook(bootPath)
	msg := &gateway.IncomingMessage{
		Platform: "cli",
		UserID:   "user1",
		Text:     "Hello",
	}

	result, err := hook.BeforeMessage(context.Background(), msg)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "System instructions here.\n\nHello", result.Text)
	assert.Equal(t, "cli", result.Platform)
	assert.Equal(t, "user1", result.UserID)
}

func TestBootMDHook_BeforeMessage_WithMissingFile(t *testing.T) {
	t.Parallel()

	hook := NewBootMDHook("/nonexistent/path/boot.md")
	msg := &gateway.IncomingMessage{
		Text: "Hello",
	}

	result, err := hook.BeforeMessage(context.Background(), msg)
	require.NoError(t, err)

	// When file is missing, the message should be returned unchanged.
	assert.Equal(t, "Hello", result.Text)
}

func TestBootMDHook_BeforeMessage_CachesContent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	bootPath := filepath.Join(tmpDir, "boot.md")
	err := os.WriteFile(bootPath, []byte("Initial content."), 0644)
	require.NoError(t, err)

	hook := NewBootMDHook(bootPath)

	// First call loads the file.
	msg1 := &gateway.IncomingMessage{Text: "msg1"}
	result1, err := hook.BeforeMessage(context.Background(), msg1)
	require.NoError(t, err)
	assert.Contains(t, result1.Text, "Initial content.")

	// Change the file on disk.
	err = os.WriteFile(bootPath, []byte("Updated content."), 0644)
	require.NoError(t, err)

	// Second call should still use cached content (loaded=true).
	msg2 := &gateway.IncomingMessage{Text: "msg2"}
	result2, err := hook.BeforeMessage(context.Background(), msg2)
	require.NoError(t, err)
	assert.Contains(t, result2.Text, "Initial content.")
	assert.NotContains(t, result2.Text, "Updated content.")
}

func TestBootMDHook_BeforeMessage_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	bootPath := filepath.Join(tmpDir, "boot.md")
	err := os.WriteFile(bootPath, []byte(""), 0644)
	require.NoError(t, err)

	hook := NewBootMDHook(bootPath)
	msg := &gateway.IncomingMessage{Text: "Hello"}

	result, err := hook.BeforeMessage(context.Background(), msg)
	require.NoError(t, err)

	// Empty content means message passes through unchanged.
	assert.Equal(t, "Hello", result.Text)
}

func TestBootMDHook_AfterMessage(t *testing.T) {
	t.Parallel()

	hook := NewBootMDHook("/nonexistent/boot.md")
	err := hook.AfterMessage(context.Background(), nil, "response text")
	assert.NoError(t, err)
}

func TestBootMDHook_DoesNotMutateOriginalMessage(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	bootPath := filepath.Join(tmpDir, "boot.md")
	err := os.WriteFile(bootPath, []byte("prefix"), 0644)
	require.NoError(t, err)

	hook := NewBootMDHook(bootPath)
	original := &gateway.IncomingMessage{Text: "original"}

	result, err := hook.BeforeMessage(context.Background(), original)
	require.NoError(t, err)

	// The original message should be unchanged.
	assert.Equal(t, "original", original.Text)
	// The result is a new message.
	assert.Contains(t, result.Text, "prefix")
}
