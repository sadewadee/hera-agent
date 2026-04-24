package byterover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetBrvCache clears the package-level lookup cache between tests so each
// test gets a fresh PATH-based resolution.
func resetBrvCache() {
	brvPathLock.Lock()
	cachedBrvPath = ""
	brvResolved = false
	brvPathLock.Unlock()
}

func TestIsAvailable_WithoutCLI(t *testing.T) {
	resetBrvCache()
	t.Setenv("PATH", "")
	t.Cleanup(resetBrvCache)

	p := New()
	assert.False(t, p.IsAvailable(), "IsAvailable should be false when brv is not on PATH")
}

func TestIsAvailable_WithMockBinary(t *testing.T) {
	resetBrvCache()
	t.Cleanup(resetBrvCache)

	dir := t.TempDir()
	mockPath := filepath.Join(dir, "brv")
	err := os.WriteFile(mockPath, []byte("#!/bin/sh\necho '{}'"), 0755)
	require.NoError(t, err)

	t.Setenv("PATH", dir)

	p := New()
	assert.True(t, p.IsAvailable(), "IsAvailable should be true when mock brv binary is on PATH")
}

func TestStoreMemory_CallsBRV(t *testing.T) {
	resetBrvCache()
	t.Cleanup(resetBrvCache)

	dir := t.TempDir()
	calledFile := filepath.Join(dir, "called.txt")
	script := "#!/bin/sh\necho \"$@\" > " + calledFile + "\necho '{}'"
	mockPath := filepath.Join(dir, "brv")
	err := os.WriteFile(mockPath, []byte(script), 0755)
	require.NoError(t, err)

	t.Setenv("PATH", dir)

	p := New()
	require.True(t, p.IsAvailable())

	// Initialize so p.cwd is set (runBrv uses it as the working directory).
	err = p.Initialize("test-session")
	require.NoError(t, err)

	// curate is the write path; HandleToolCall exercises runBrv directly.
	_, err = p.HandleToolCall("brv_curate", map[string]interface{}{
		"content": "test fact about Go",
	})
	require.NoError(t, err)

	called, err := os.ReadFile(calledFile)
	require.NoError(t, err)
	assert.Contains(t, string(called), "test fact about Go",
		"brv binary should have received the content as an argument")
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	p := New()
	_, err := p.HandleToolCall("nonexistent_tool", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
