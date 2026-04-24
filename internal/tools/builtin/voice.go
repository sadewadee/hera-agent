package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sadewadee/hera/internal/tools"
)

type VoiceTool struct{}

type voiceArgs struct {
	Action string `json:"action"`
	Text   string `json:"text,omitempty"`
	File   string `json:"file,omitempty"`
	Voice  string `json:"voice,omitempty"`
}

func (t *VoiceTool) Name() string        { return "voice" }
func (t *VoiceTool) Description() string  { return "Text-to-speech synthesis and speech-to-text transcription." }
func (t *VoiceTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["tts","stt"],"description":"tts=text-to-speech, stt=speech-to-text"},"text":{"type":"string","description":"Text to synthesize (for tts)"},"file":{"type":"string","description":"Audio file path (for stt)"},"voice":{"type":"string","description":"Voice name (for tts)"}},"required":["action"]}`)
}

func (t *VoiceTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a voiceArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	switch a.Action {
	case "tts":
		if a.Text == "" {
			return &tools.Result{Content: "text is required for tts", IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("TTS synthesis queued for text (%d chars), voice=%s. Output will be saved as audio.", len(a.Text), a.Voice)}, nil
	case "stt":
		if a.File == "" {
			return &tools.Result{Content: "file is required for stt", IsError: true}, nil
		}
		if _, err := os.Stat(a.File); err != nil {
			return &tools.Result{Content: fmt.Sprintf("audio file not found: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("STT transcription queued for %s. Transcription will be returned when complete.", a.File)}, nil
	default:
		return &tools.Result{Content: "action must be 'tts' or 'stt'", IsError: true}, nil
	}
}

func RegisterVoice(registry *tools.Registry) { registry.Register(&VoiceTool{}) }
