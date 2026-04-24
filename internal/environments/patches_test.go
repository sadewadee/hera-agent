package environments

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchOperation_Apply(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("original"), 0o644))

	patch := PatchOperation{
		FilePath: filePath,
		Original: []byte("original"),
		Modified: []byte("modified"),
	}

	require.NoError(t, patch.Apply())

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "modified", string(data))
}

func TestPatchOperation_Revert(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("modified"), 0o644))

	patch := PatchOperation{
		FilePath: filePath,
		Original: []byte("original"),
		Modified: []byte("modified"),
	}

	require.NoError(t, patch.Revert())

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

func TestPatchOperation_ApplyAndRevert(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("original"), 0o644))

	patch := PatchOperation{
		FilePath: filePath,
		Original: []byte("original"),
		Modified: []byte("modified content here"),
	}

	require.NoError(t, patch.Apply())
	data, _ := os.ReadFile(filePath)
	assert.Equal(t, "modified content here", string(data))

	require.NoError(t, patch.Revert())
	data, _ = os.ReadFile(filePath)
	assert.Equal(t, "original", string(data))
}

func TestParseUnifiedDiff(t *testing.T) {
	diff := `--- a/file1.go
+++ b/file1.go
@@ -1,3 +1,3 @@
-old line
+new line
--- a/file2.txt
+++ b/file2.txt
@@ -1 +1 @@
-old
+new`

	patches := ParseUnifiedDiff(diff)
	assert.Len(t, patches, 2)
	assert.Equal(t, "file1.go", patches[0].FilePath)
	assert.Equal(t, "file2.txt", patches[1].FilePath)
}

func TestParseUnifiedDiff_Empty(t *testing.T) {
	patches := ParseUnifiedDiff("")
	assert.Empty(t, patches)
}

func TestParseUnifiedDiff_NoValidPairs(t *testing.T) {
	diff := "some random text\nno diff markers here"
	patches := ParseUnifiedDiff(diff)
	assert.Empty(t, patches)
}
