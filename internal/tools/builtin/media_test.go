package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageGenerateTool_Name(t *testing.T) {
	tool := &ImageGenerateTool{}
	assert.Equal(t, "image_generate", tool.Name())
}

func TestImageGenerateTool_Description(t *testing.T) {
	tool := &ImageGenerateTool{}
	assert.Contains(t, tool.Description(), "image")
}

func TestImageGenerateTool_InvalidArgs(t *testing.T) {
	tool := &ImageGenerateTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestImageGenerateTool_EmptyPrompt(t *testing.T) {
	tool := &ImageGenerateTool{}
	args, _ := json.Marshal(imageGenArgs{Prompt: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "prompt is required")
}

func TestImageGenerateTool_NoAPIKey(t *testing.T) {
	tool := &ImageGenerateTool{apiKey: ""}
	args, _ := json.Marshal(imageGenArgs{Prompt: "a cat"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "FAL_KEY")
}

func TestTextToSpeechTool_Name(t *testing.T) {
	tool := &TextToSpeechTool{}
	assert.Equal(t, "text_to_speech", tool.Name())
}

func TestTextToSpeechTool_Description(t *testing.T) {
	tool := &TextToSpeechTool{}
	assert.Contains(t, tool.Description(), "speech")
}

func TestTextToSpeechTool_InvalidArgs(t *testing.T) {
	tool := &TextToSpeechTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTextToSpeechTool_EmptyText(t *testing.T) {
	tool := &TextToSpeechTool{}
	args, _ := json.Marshal(ttsArgs{Text: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "text is required")
}

func TestTextToSpeechTool_NoAPIKey(t *testing.T) {
	tool := &TextToSpeechTool{apiKey: ""}
	args, _ := json.Marshal(ttsArgs{Text: "Hello world"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "ELEVENLABS_API_KEY")
}

func TestTruncateMedia(t *testing.T) {
	assert.Equal(t, "short", truncateMedia("short", 100))
	assert.Equal(t, "abc...", truncateMedia("abcdef", 3))
}
