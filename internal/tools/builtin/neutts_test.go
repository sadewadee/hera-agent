package builtin

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NeuTTSTool interface ---

func TestNeuTTSTool_Name(t *testing.T) {
	tool := &NeuTTSTool{outputDir: "/tmp", device: "cpu", model: "test"}
	assert.Equal(t, "neutts_synth", tool.Name())
}

func TestNeuTTSTool_Description(t *testing.T) {
	tool := &NeuTTSTool{}
	assert.Contains(t, tool.Description(), "NeuTTS")
}

func TestNeuTTSTool_Parameters(t *testing.T) {
	tool := &NeuTTSTool{}
	params := tool.Parameters()
	assert.NotNil(t, params)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(params, &schema))
	props := schema["properties"].(map[string]any)
	assert.Contains(t, props, "text")
	assert.Contains(t, props, "ref_audio")
	assert.Contains(t, props, "ref_text")
}

func TestNeuTTSTool_Execute_EmptyText(t *testing.T) {
	tool := &NeuTTSTool{outputDir: t.TempDir(), device: "cpu", model: "test"}
	args, _ := json.Marshal(map[string]string{
		"text":      "",
		"ref_audio": "/tmp/ref.wav",
		"ref_text":  "/tmp/ref.txt",
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "text is required")
}

func TestNeuTTSTool_Execute_MissingRefAudio(t *testing.T) {
	tool := &NeuTTSTool{outputDir: t.TempDir(), device: "cpu", model: "test"}
	args, _ := json.Marshal(map[string]string{
		"text":      "hello",
		"ref_audio": "",
		"ref_text":  "/tmp/ref.txt",
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "ref_audio is required")
}

func TestNeuTTSTool_Execute_MissingRefText(t *testing.T) {
	tool := &NeuTTSTool{outputDir: t.TempDir(), device: "cpu", model: "test"}
	args, _ := json.Marshal(map[string]string{
		"text":      "hello",
		"ref_audio": "/tmp/ref.wav",
		"ref_text":  "",
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "ref_text is required")
}

func TestNeuTTSTool_Execute_RefAudioNotFound(t *testing.T) {
	tool := &NeuTTSTool{outputDir: t.TempDir(), device: "cpu", model: "test"}
	args, _ := json.Marshal(map[string]string{
		"text":      "hello",
		"ref_audio": "/nonexistent/audio.wav",
		"ref_text":  "/nonexistent/text.txt",
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "reference audio not found")
}

func TestNeuTTSTool_Execute_InvalidArgs(t *testing.T) {
	tool := &NeuTTSTool{outputDir: t.TempDir(), device: "cpu", model: "test"}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid arguments")
}

// --- WriteWAV ---

func TestWriteWAV_Success(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "test.wav")

	samples := make([]float32, 48000)
	for i := range samples {
		samples[i] = float32(math.Sin(2 * math.Pi * 440 * float64(i) / 24000))
	}

	err := WriteWAV(outPath, samples, 24000)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	// Check WAV header.
	assert.Equal(t, "RIFF", string(data[0:4]))
	assert.Equal(t, "WAVE", string(data[8:12]))
	assert.Equal(t, "fmt ", string(data[12:16]))
	assert.Equal(t, "data", string(data[36:40]))

	// Check sample rate.
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	assert.Equal(t, uint32(24000), sampleRate)
}

func TestWriteWAV_DefaultSampleRate(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "test.wav")

	err := WriteWAV(outPath, []float32{0.0, 0.5, -0.5}, 0)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	assert.Equal(t, uint32(24000), sampleRate) // default
}

func TestWriteWAV_EmptySamples(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "empty.wav")

	err := WriteWAV(outPath, nil, 24000)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, "RIFF", string(data[0:4]))
}

func TestWriteWAV_Clamping(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "clamped.wav")

	// Values beyond [-1,1] should be clamped.
	samples := []float32{-5.0, 5.0, 0.0}
	err := WriteWAV(outPath, samples, 16000)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	// Read first PCM sample (offset 44 for standard WAV).
	pcm1 := int16(binary.LittleEndian.Uint16(data[44:46]))
	assert.Equal(t, int16(-32767), pcm1) // clamped to -1.0

	pcm2 := int16(binary.LittleEndian.Uint16(data[46:48]))
	assert.Equal(t, int16(32767), pcm2) // clamped to 1.0
}
