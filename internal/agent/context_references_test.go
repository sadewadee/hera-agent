package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandContextReferences_NoReferences(t *testing.T) {
	text := "Hello world, this is a plain message."
	result := ExpandContextReferences(text)
	assert.Equal(t, text, result)
}

func TestExpandContextReferences_FileReference(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "sample.txt")
	require.NoError(t, os.WriteFile(fpath, []byte("file content here"), 0644))

	text := "Check this: @file:" + fpath
	result := ExpandContextReferences(text)
	assert.Contains(t, result, "file content here")
	assert.Contains(t, result, "<file path=")
}

func TestExpandContextReferences_FileNotFound(t *testing.T) {
	text := "Check this: @file:/nonexistent/path.txt"
	result := ExpandContextReferences(text)
	assert.Contains(t, result, "[error reading")
}

func TestExpandContextReferences_URLReference(t *testing.T) {
	text := "See @url:https://example.com"
	result := ExpandContextReferences(text)
	assert.Contains(t, result, "URL content")
	assert.Contains(t, result, "example.com")
}

func TestExpandContextReferences_LargeFileIsTruncated(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "large.txt")
	// Create content > 50000 chars
	data := make([]byte, 60000)
	for i := range data {
		data[i] = 'x'
	}
	require.NoError(t, os.WriteFile(fpath, data, 0644))

	text := "@file:" + fpath
	result := ExpandContextReferences(text)
	assert.Contains(t, result, "[truncated]")
}

func TestExpandContextReferences_MultipleReferences(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(f1, []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(f2, []byte("bbb"), 0644))

	text := "@file:" + f1 + " and @file:" + f2
	result := ExpandContextReferences(text)
	assert.Contains(t, result, "aaa")
	assert.Contains(t, result, "bbb")
}

func TestExpandContextReferences_RefPatternMatching(t *testing.T) {
	assert.True(t, refPattern.MatchString("@file:/path/to/file.go"))
	assert.True(t, refPattern.MatchString("@diff:HEAD~1"))
	assert.True(t, refPattern.MatchString("@staged:all"))
	assert.True(t, refPattern.MatchString("@url:https://example.com"))
	assert.False(t, refPattern.MatchString("@unknown:something"))
}
