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

func TestVisionTool_Name(t *testing.T) {
	tool := &VisionTool{}
	assert.Equal(t, "vision", tool.Name())
}

func TestVisionTool_Description(t *testing.T) {
	tool := &VisionTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestVisionTool_InvalidArgs(t *testing.T) {
	tool := &VisionTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestVisionTool_NoImageProvided(t *testing.T) {
	tool := &VisionTool{}
	args, _ := json.Marshal(visionArgs{Prompt: "What is this?"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "image_url or image_path is required")
}

func TestVisionTool_ImagePathNotFound(t *testing.T) {
	tool := &VisionTool{}
	args, _ := json.Marshal(visionArgs{ImagePath: "/nonexistent/image.png", Prompt: "What?"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "image not found")
}

func TestVisionTool_NoAPIKey(t *testing.T) {
	tool := &VisionTool{apiKey: ""}
	// Ensure env var is not set for this test
	origKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("OPENAI_API_KEY", origKey)
		}
	}()

	args, _ := json.Marshal(visionArgs{ImageURL: "https://example.com/img.png", Prompt: "What?"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "API key not configured")
}

func TestVisionTool_WithAPIKey(t *testing.T) {
	tool := &VisionTool{apiKey: "test-key"}
	args, _ := json.Marshal(visionArgs{ImageURL: "https://example.com/img.png", Prompt: "Describe"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "vision tool must return error when multimodal LLM not wired")
	assert.Contains(t, result.Content, "not yet wired")
}

func TestVisionTool_WithImagePath(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "test.png")
	os.WriteFile(imgPath, []byte("fake-image"), 0644)

	tool := &VisionTool{apiKey: "test-key"}
	args, _ := json.Marshal(visionArgs{ImagePath: imgPath, Prompt: "What is this?"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "vision tool must return error when multimodal LLM not wired")
	assert.Contains(t, result.Content, "not yet wired")
}
