// Package builtin provides built-in tool implementations.
//
// image_gen.go implements the image generation tool that dispatches to
// various image generation APIs (DALL-E, Stable Diffusion, etc.)
// based on configuration.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// ImageGenTool generates images from text prompts.
type ImageGenTool struct {
	apiKey     string
	provider   string
	httpClient *http.Client
	outputDir  string
}

// RegisterImageGen registers the image generation tool.
func RegisterImageGen(registry *tools.Registry) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	provider := os.Getenv("IMAGE_GEN_PROVIDER")
	if provider == "" {
		provider = "dall-e"
	}

	outputDir := filepath.Join(paths.HeraHome(), "generated_images")

	tool := &ImageGenTool{
		apiKey:   apiKey,
		provider: provider,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		outputDir: outputDir,
	}
	registry.Register(tool)
}

func (t *ImageGenTool) Name() string { return "image_generation" }
func (t *ImageGenTool) Description() string {
	return "Generate images from text prompts using AI models"
}

type imageGenToolArgs struct {
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
	Style  string `json:"style,omitempty"`
	N      int    `json:"n,omitempty"`
}

func (t *ImageGenTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"prompt": {"type": "string", "description": "Text description of the image to generate"},
			"size": {"type": "string", "description": "Image size (e.g., 1024x1024, 512x512)"},
			"style": {"type": "string", "description": "Image style (vivid, natural)"},
			"n": {"type": "integer", "description": "Number of images to generate (1-4)"}
		},
		"required": ["prompt"]
	}`)
}

func (t *ImageGenTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a imageGenToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if a.Prompt == "" {
		return &tools.Result{Content: "prompt is required", IsError: true}, nil
	}

	size := a.Size
	if size == "" {
		size = "1024x1024"
	}

	style := a.Style
	if style == "" {
		style = "vivid"
	}

	if t.apiKey == "" {
		return &tools.Result{Content: "OPENAI_API_KEY not set for image generation", IsError: true}, nil
	}

	slog.Info("generating image",
		"prompt_len", len(a.Prompt),
		"size", size,
		"provider", t.provider,
	)

	// Call DALL-E API.
	payload := map[string]any{
		"model":  "dall-e-3",
		"prompt": a.Prompt,
		"n":      1,
		"size":   size,
		"style":  style,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/images/generations",
		strings.NewReader(string(body)))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("request error: %v", err), IsError: true}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("API error: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &tools.Result{Content: fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(respBody)), IsError: true}, nil
	}

	var result struct {
		Data []struct {
			URL           string `json:"url"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &tools.Result{Content: fmt.Sprintf("parse response: %v", err), IsError: true}, nil
	}

	if len(result.Data) == 0 {
		return &tools.Result{Content: "no images returned", IsError: true}, nil
	}

	// Save URL reference.
	_ = os.MkdirAll(t.outputDir, 0o755)
	imgURL := result.Data[0].URL
	revised := result.Data[0].RevisedPrompt

	output := fmt.Sprintf("Image generated successfully.\nURL: %s", imgURL)
	if revised != "" {
		output += fmt.Sprintf("\nRevised prompt: %s", revised)
	}
	return &tools.Result{Content: output}, nil
}
