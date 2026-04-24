package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// ImageGenerateTool generates images using the FAL AI API.
type ImageGenerateTool struct {
	apiKey string
	client *http.Client
}

type imageGenArgs struct {
	Prompt   string `json:"prompt"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	NumSteps int    `json:"num_steps,omitempty"`
}

func (t *ImageGenerateTool) Name() string {
	return "image_generate"
}

func (t *ImageGenerateTool) Description() string {
	return "Generates an image from a text prompt using FAL AI (Flux model). Returns the image URL."
}

func (t *ImageGenerateTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"prompt": {
				"type": "string",
				"description": "Text description of the image to generate."
			},
			"width": {
				"type": "integer",
				"description": "Image width in pixels. Defaults to 1024."
			},
			"height": {
				"type": "integer",
				"description": "Image height in pixels. Defaults to 1024."
			},
			"num_steps": {
				"type": "integer",
				"description": "Number of inference steps. Higher = better quality. Defaults to 28."
			}
		},
		"required": ["prompt"]
	}`)
}

func (t *ImageGenerateTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params imageGenArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Prompt == "" {
		return &tools.Result{Content: "prompt is required", IsError: true}, nil
	}

	if t.apiKey == "" {
		return &tools.Result{
			Content: "image_generate is not configured: set FAL_KEY environment variable",
			IsError: true,
		}, nil
	}

	width := params.Width
	if width <= 0 {
		width = 1024
	}
	height := params.Height
	if height <= 0 {
		height = 1024
	}
	numSteps := params.NumSteps
	if numSteps <= 0 {
		numSteps = 28
	}

	reqBody := map[string]any{
		"prompt":                params.Prompt,
		"image_size":            map[string]int{"width": width, "height": height},
		"num_inference_steps":   numSteps,
		"num_images":            1,
		"enable_safety_checker": true,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("marshal request: %v", err), IsError: true}, nil
	}

	// POST to FAL AI queue endpoint.
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://queue.fal.run/fal-ai/flux/dev", bytes.NewReader(data))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+t.apiKey)

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("read response: %v", err), IsError: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &tools.Result{
			Content: fmt.Sprintf("FAL AI error (status %d): %s", resp.StatusCode, truncateMedia(string(body), 500)),
			IsError: true,
		}, nil
	}

	// Parse response -- FAL returns either direct result or a request_id for polling.
	var falResp map[string]any
	if err := json.Unmarshal(body, &falResp); err != nil {
		return &tools.Result{Content: fmt.Sprintf("parse response: %v", err), IsError: true}, nil
	}

	// Check for direct images in response.
	if images, ok := falResp["images"].([]any); ok && len(images) > 0 {
		if img, ok := images[0].(map[string]any); ok {
			if url, ok := img["url"].(string); ok {
				return &tools.Result{Content: fmt.Sprintf("Image generated successfully!\n\nURL: %s\nPrompt: %s", url, params.Prompt)}, nil
			}
		}
	}

	// Check for queued response.
	if reqID, ok := falResp["request_id"].(string); ok {
		return &tools.Result{
			Content: fmt.Sprintf("Image generation queued (request_id: %s). The image is being generated and will be available shortly.", reqID),
		}, nil
	}

	return &tools.Result{Content: fmt.Sprintf("Unexpected FAL response: %s", truncateMedia(string(body), 500))}, nil
}

// TextToSpeechTool converts text to speech audio.
type TextToSpeechTool struct {
	apiKey   string
	provider string // "elevenlabs" or "edge"
	client   *http.Client
}

type ttsArgs struct {
	Text     string `json:"text"`
	Voice    string `json:"voice,omitempty"`
	Language string `json:"language,omitempty"`
}

func (t *TextToSpeechTool) Name() string {
	return "text_to_speech"
}

func (t *TextToSpeechTool) Description() string {
	return "Converts text to speech audio using ElevenLabs API. Returns the audio file path or URL."
}

func (t *TextToSpeechTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"text": {
				"type": "string",
				"description": "The text to convert to speech."
			},
			"voice": {
				"type": "string",
				"description": "Voice ID or name. Defaults to 'Rachel'."
			},
			"language": {
				"type": "string",
				"description": "Language code (e.g., 'en', 'es'). Defaults to 'en'."
			}
		},
		"required": ["text"]
	}`)
}

func (t *TextToSpeechTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params ttsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Text == "" {
		return &tools.Result{Content: "text is required", IsError: true}, nil
	}

	if t.apiKey == "" {
		return &tools.Result{
			Content: "text_to_speech is not configured: set ELEVENLABS_API_KEY environment variable",
			IsError: true,
		}, nil
	}

	voice := params.Voice
	if voice == "" {
		voice = "21m00Tcm4TlvDq8ikWAM" // Rachel voice ID
	}

	// POST to ElevenLabs API.
	reqBody := map[string]any{
		"text":     params.Text,
		"model_id": "eleven_monolingual_v1",
		"voice_settings": map[string]any{
			"stability":        0.5,
			"similarity_boost": 0.75,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("marshal request: %v", err), IsError: true}, nil
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voice)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", t.apiKey)
	httpReq.Header.Set("Accept", "audio/mpeg")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &tools.Result{
			Content: fmt.Sprintf("ElevenLabs error (status %d): %s", resp.StatusCode, truncateMedia(string(body), 500)),
			IsError: true,
		}, nil
	}

	// Save audio to temp file.
	tmpFile, err := os.CreateTemp("", "hera-tts-*.mp3")
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create temp file: %v", err), IsError: true}, nil
	}
	defer tmpFile.Close()

	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("save audio: %v", err), IsError: true}, nil
	}

	return &tools.Result{
		Content: fmt.Sprintf("Audio generated successfully!\n\nFile: %s\nSize: %d bytes\nText: %s",
			tmpFile.Name(), written, truncateMedia(params.Text, 100)),
	}, nil
}

// RegisterMedia registers image_generate and text_to_speech tools.
func RegisterMedia(registry *tools.Registry) {
	httpClient := &http.Client{Timeout: 120 * time.Second}
	falKey := os.Getenv("FAL_KEY")
	elevenKey := os.Getenv("ELEVENLABS_API_KEY")

	registry.Register(&ImageGenerateTool{apiKey: falKey, client: httpClient})
	registry.Register(&TextToSpeechTool{apiKey: elevenKey, provider: "elevenlabs", client: httpClient})
}

func truncateMedia(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
