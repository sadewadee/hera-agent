package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sadewadee/hera/internal/tools"
)

// VisionTool analyzes images via a multimodal LLM provider.
// Requires OPENAI_API_KEY and an OpenAI GPT-4o or Anthropic Claude 3+ provider.
// The multimodal LLM call is not yet wired; the tool returns an explicit error
// until a vision provider integration is added.
type VisionTool struct{ apiKey string }

type visionArgs struct {
	ImageURL  string `json:"image_url,omitempty"`
	ImagePath string `json:"image_path,omitempty"`
	Prompt    string `json:"prompt"`
}

func (t *VisionTool) Name() string        { return "vision" }
func (t *VisionTool) Description() string { return "Analyzes images and answers questions about them." }
func (t *VisionTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"image_url":{"type":"string","description":"URL of image to analyze"},"image_path":{"type":"string","description":"Local path to image"},"prompt":{"type":"string","description":"Question about the image"}},"required":["prompt"]}`)
}

func (t *VisionTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	_ = ctx
	var a visionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	if a.ImageURL == "" && a.ImagePath == "" {
		return &tools.Result{Content: "either image_url or image_path is required", IsError: true}, nil
	}
	if a.ImagePath != "" {
		if _, err := os.Stat(a.ImagePath); err != nil {
			return &tools.Result{Content: fmt.Sprintf("image not found: %v", err), IsError: true}, nil
		}
	}
	if t.apiKey == "" {
		t.apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if t.apiKey == "" {
		return &tools.Result{Content: "vision API key not configured (set OPENAI_API_KEY)", IsError: true}, nil
	}
	return &tools.Result{
		Content: "vision: multimodal LLM call not yet wired; configure an OpenAI GPT-4o or Anthropic Claude 3+ provider",
		IsError: true,
	}, nil
}

func RegisterVision(registry *tools.Registry) { registry.Register(&VisionTool{}) }
