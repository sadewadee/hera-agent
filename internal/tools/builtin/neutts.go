// Package builtin provides built-in tool implementations.
//
// neutts.go implements text-to-speech synthesis by invoking an external
// NeuTTS process. The TTS model is kept in a separate subprocess to
// avoid lingering memory after synthesis completes.
package builtin

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// NeuTTSTool generates speech audio from text using NeuTTS.
type NeuTTSTool struct {
	outputDir string
	device    string
	model     string
}

// RegisterNeuTTS registers the NeuTTS tool with the registry.
func RegisterNeuTTS(registry *tools.Registry) {
	outputDir := filepath.Join(paths.HeraHome(), "tts_output")

	device := os.Getenv("NEUTTS_DEVICE")
	if device == "" {
		device = "cpu"
	}
	model := os.Getenv("NEUTTS_MODEL")
	if model == "" {
		model = "neuphonic/neutts-air-q4-gguf"
	}

	tool := &NeuTTSTool{
		outputDir: outputDir,
		device:    device,
		model:     model,
	}
	registry.Register(tool)
}

func (t *NeuTTSTool) Name() string { return "neutts_synth" }
func (t *NeuTTSTool) Description() string {
	return "Synthesize speech from text using NeuTTS voice cloning"
}

func (t *NeuTTSTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"text":      {"type": "string", "description": "Text to synthesize into speech"},
			"ref_audio": {"type": "string", "description": "Path to reference voice audio file"},
			"ref_text":  {"type": "string", "description": "Path to reference voice transcript file"},
			"output":    {"type": "string", "description": "Output WAV file path"}
		},
		"required": ["text", "ref_audio", "ref_text"]
	}`)
}

func (t *NeuTTSTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params struct {
		Text     string `json:"text"`
		RefAudio string `json:"ref_audio"`
		RefText  string `json:"ref_text"`
		Output   string `json:"output"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	if params.Text == "" {
		return &tools.Result{Content: "text is required", IsError: true}, nil
	}
	if params.RefAudio == "" {
		return &tools.Result{Content: "ref_audio is required", IsError: true}, nil
	}
	if params.RefText == "" {
		return &tools.Result{Content: "ref_text is required", IsError: true}, nil
	}

	// Normalize user-supplied paths (~, $HERA_HOME, .hera/…).
	params.RefAudio = paths.Normalize(params.RefAudio)
	params.RefText = paths.Normalize(params.RefText)
	params.Output = paths.Normalize(params.Output)

	// Validate reference files exist.
	if _, err := os.Stat(params.RefAudio); err != nil {
		return &tools.Result{Content: fmt.Sprintf("reference audio not found: %s", params.RefAudio), IsError: true}, nil
	}
	if _, err := os.Stat(params.RefText); err != nil {
		return &tools.Result{Content: fmt.Sprintf("reference text not found: %s", params.RefText), IsError: true}, nil
	}

	// Determine output path.
	outPath := params.Output
	if outPath == "" {
		_ = os.MkdirAll(t.outputDir, 0o755)
		outPath = filepath.Join(t.outputDir, "synthesis.wav")
	}
	outDir := filepath.Dir(outPath)
	_ = os.MkdirAll(outDir, 0o755)

	slog.Info("starting NeuTTS synthesis",
		"text_len", len(params.Text),
		"device", t.device,
		"output", outPath,
	)

	// Invoke external Python NeuTTS process.
	cmd := exec.CommandContext(ctx, "python", "-m", "tools.neutts_synth",
		"--text", params.Text,
		"--out", outPath,
		"--ref-audio", params.RefAudio,
		"--ref-text", params.RefText,
		"--model", t.model,
		"--device", t.device,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &tools.Result{
			Content: fmt.Sprintf("NeuTTS synthesis failed: %v\n%s", err, string(output)),
			IsError: true,
		}, nil
	}

	return &tools.Result{Content: fmt.Sprintf("Speech synthesized successfully: %s", outPath)}, nil
}

// WriteWAV writes mono 16-bit PCM WAV data from float32 samples.
// This is a Go port of the Python _write_wav helper for use by other
// tools that need to generate WAV files without external dependencies.
func WriteWAV(path string, samples []float32, sampleRate int) error {
	if sampleRate <= 0 {
		sampleRate = 24000
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create WAV file: %w", err)
	}
	defer f.Close()

	numChannels := uint16(1)
	bitsPerSample := uint16(16)
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample/8)
	blockAlign := numChannels * (bitsPerSample / 8)
	dataSize := uint32(len(samples)) * uint32(bitsPerSample/8)

	// RIFF header.
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte("WAVE"))

	// fmt chunk.
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(f, binary.LittleEndian, numChannels)
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	binary.Write(f, binary.LittleEndian, byteRate)
	binary.Write(f, binary.LittleEndian, blockAlign)
	binary.Write(f, binary.LittleEndian, bitsPerSample)

	// data chunk.
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, dataSize)

	for _, s := range samples {
		// Clamp to [-1.0, 1.0] and convert to int16.
		clamped := math.Max(-1.0, math.Min(1.0, float64(s)))
		pcm := int16(clamped * 32767)
		binary.Write(f, binary.LittleEndian, pcm)
	}

	return nil
}

func init() {
	_ = strings.TrimSpace // avoid unused import
}
