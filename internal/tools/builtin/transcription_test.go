package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranscriptionTool_Name(t *testing.T) {
	tool := &TranscriptionTool{}
	assert.Equal(t, "transcription", tool.Name())
}

func TestTranscriptionTool_Description(t *testing.T) {
	tool := &TranscriptionTool{}
	assert.Contains(t, tool.Description(), "Transcribes")
}

func TestTranscriptionTool_InvalidArgs(t *testing.T) {
	tool := &TranscriptionTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTranscriptionTool_FileNotFound(t *testing.T) {
	tool := &TranscriptionTool{}
	args, _ := json.Marshal(transcriptionArgs{FilePath: "/nonexistent/audio.wav"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "file not found")
}

func TestTranscriptionTool_WithFile(t *testing.T) {
	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("fake audio data"), 0o644))

	tool := &TranscriptionTool{}
	args, _ := json.Marshal(transcriptionArgs{FilePath: audioFile})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "en")
}

func TestTranscriptionTool_CustomLanguage(t *testing.T) {
	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.mp3")
	require.NoError(t, os.WriteFile(audioFile, []byte("fake"), 0o644))

	tool := &TranscriptionTool{}
	args, _ := json.Marshal(transcriptionArgs{FilePath: audioFile, Language: "es"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "es")
}

func TestRegisterTranscription(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTranscription(registry)
	_, ok := registry.Get("transcription")
	assert.True(t, ok)
}
