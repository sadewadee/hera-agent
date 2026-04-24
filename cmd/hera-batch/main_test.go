package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/batch"
)

// TestMakeWriter_JSONL verifies that makeWriter with format=jsonl returns a JSONLWriter.
func TestMakeWriter_JSONL(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.jsonl")

	w, err := makeWriter("jsonl", out)
	if err != nil {
		t.Fatalf("makeWriter: %v", err)
	}
	defer w.Close()

	r := batch.PromptResult{Index: 0, Prompt: "hello", Response: "world"}
	if err := w.Write(r); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = w.Close()

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(data), `"prompt":"hello"`) {
		t.Errorf("expected prompt in output: %s", string(data))
	}
}

func TestMakeWriter_CSV(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.csv")

	w, err := makeWriter("csv", out)
	if err != nil {
		t.Fatalf("makeWriter: %v", err)
	}

	r := batch.PromptResult{Index: 1, Prompt: "q", Response: "a"}
	if err := w.Write(r); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = w.Close()

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	// CSV should have header row.
	if !strings.Contains(string(data), "index") {
		t.Errorf("expected CSV header: %s", string(data))
	}
}

func TestMakeWriter_Text(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")

	w, err := makeWriter("text", out)
	if err != nil {
		t.Fatalf("makeWriter: %v", err)
	}

	r := batch.PromptResult{Index: 2, Prompt: "ask", Response: "ans"}
	if err := w.Write(r); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = w.Close()

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(data), "[2]") {
		t.Errorf("expected index [2] in output: %s", string(data))
	}
}

func TestMakeWriter_Stdout(t *testing.T) {
	// When no output file is specified, writer should write to stdout (no error).
	w, err := makeWriter("jsonl", "")
	if err != nil {
		t.Fatalf("makeWriter stdout: %v", err)
	}
	// Should not close stdout on Close().
	_ = w.Close()
}

func TestMakeWriter_InvalidPath(t *testing.T) {
	_, err := makeWriter("jsonl", "/nonexistent-dir/impossible/path/file.jsonl")
	if err == nil {
		t.Error("expected error for unwritable path, got nil")
	}
}
