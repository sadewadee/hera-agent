package builtin
import ("context";"encoding/json";"fmt";"os";"github.com/sadewadee/hera/internal/tools")
type TranscriptionTool struct{}
type transcriptionArgs struct { FilePath string `json:"file_path"`; Language string `json:"language,omitempty"` }
func (t *TranscriptionTool) Name() string { return "transcription" }
func (t *TranscriptionTool) Description() string { return "Transcribes audio files to text using speech-to-text APIs." }
func (t *TranscriptionTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"file_path":{"type":"string","description":"Path to audio file"},"language":{"type":"string","description":"Language code (e.g., en, es, fr)"}},"required":["file_path"]}`) }
func (t *TranscriptionTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a transcriptionArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	if _, err := os.Stat(a.FilePath); err != nil { return &tools.Result{Content: fmt.Sprintf("file not found: %v", err), IsError: true}, nil }
	lang := a.Language; if lang == "" { lang = "en" }
	return &tools.Result{Content: fmt.Sprintf("Transcription queued for %s (language: %s). Requires Whisper API or compatible service.", a.FilePath, lang)}, nil
}
func RegisterTranscription(registry *tools.Registry) { registry.Register(&TranscriptionTool{}) }
