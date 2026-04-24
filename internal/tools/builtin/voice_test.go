package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVoiceTool_Name(t *testing.T) {
	tool := &VoiceTool{}
	assert.Equal(t, "voice", tool.Name())
}

func TestVoiceTool_Description(t *testing.T) {
	tool := &VoiceTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestVoiceTool_InvalidArgs(t *testing.T) {
	tool := &VoiceTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestVoiceTool_TTS_Success(t *testing.T) {
	tool := &VoiceTool{}
	args, _ := json.Marshal(voiceArgs{Action: "tts", Text: "Hello world", Voice: "alloy"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "TTS synthesis")
	assert.Contains(t, result.Content, "11 chars")
	assert.Contains(t, result.Content, "alloy")
}

func TestVoiceTool_TTS_NoText(t *testing.T) {
	tool := &VoiceTool{}
	args, _ := json.Marshal(voiceArgs{Action: "tts"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "text is required")
}

func TestVoiceTool_STT_Success(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.wav")
	os.WriteFile(audioPath, []byte("fake-audio"), 0644)

	tool := &VoiceTool{}
	args, _ := json.Marshal(voiceArgs{Action: "stt", File: audioPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "STT transcription")
}

func TestVoiceTool_STT_NoFile(t *testing.T) {
	tool := &VoiceTool{}
	args, _ := json.Marshal(voiceArgs{Action: "stt"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "file is required")
}

func TestVoiceTool_STT_FileNotFound(t *testing.T) {
	tool := &VoiceTool{}
	args, _ := json.Marshal(voiceArgs{Action: "stt", File: "/nonexistent/audio.wav"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "audio file not found")
}

func TestVoiceTool_InvalidAction(t *testing.T) {
	tool := &VoiceTool{}
	args, _ := json.Marshal(voiceArgs{Action: "unknown"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "tts' or 'stt")
}
