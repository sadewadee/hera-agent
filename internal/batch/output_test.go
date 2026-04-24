package batch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func makeResult(index int, prompt, response string, dur time.Duration) PromptResult {
	return PromptResult{
		Index:    index,
		Prompt:   prompt,
		Response: response,
		Duration: dur,
	}
}

func makeErrResult(index int, prompt string, err error) PromptResult {
	return PromptResult{
		Index:    index,
		Prompt:   prompt,
		Err:      err,
		Duration: 100 * time.Millisecond,
	}
}

// ---------- JSONLWriter ----------

func TestJSONLWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONLWriter(&buf)

	r := makeResult(0, "hello", "world", 42*time.Millisecond)
	if err := w.Write(r); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	var rec JSONLRecord
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("unmarshal: %v — raw: %q", err, line)
	}
	if rec.Index != 0 || rec.Prompt != "hello" || rec.Response != "world" {
		t.Errorf("unexpected record: %+v", rec)
	}
	if rec.DurationMs != 42 {
		t.Errorf("duration_ms: got %d, want 42", rec.DurationMs)
	}
}

func TestJSONLWriter_Error(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONLWriter(&buf)

	r := makeErrResult(1, "q", fmt.Errorf("boom"))
	if err := w.Write(r); err != nil {
		t.Fatalf("Write error result: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	var rec JSONLRecord
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec.Error != "boom" {
		t.Errorf("error field: got %q, want %q", rec.Error, "boom")
	}
}

func TestJSONLWriter_MultipleLines(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONLWriter(&buf)
	for i := 0; i < 3; i++ {
		if err := w.Write(makeResult(i, "p", "r", time.Millisecond)); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

// ---------- CSVWriter ----------

func TestCSVWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	w := NewCSVWriter(&buf)

	if err := w.Write(makeResult(0, "p1", "r1", 10*time.Millisecond)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Write(makeResult(1, "p2", "r2", 20*time.Millisecond)); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	_ = w.Close()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// header + 2 data rows
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "index") {
		t.Errorf("first line should be header, got %q", lines[0])
	}
}

// ---------- TextWriter ----------

func TestTextWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	w := NewTextWriter(&buf)

	if err := w.Write(makeResult(7, "my-prompt", "my-response", time.Second)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[7]") {
		t.Errorf("expected index [7] in output: %q", out)
	}
	if !strings.Contains(out, "my-prompt") {
		t.Errorf("expected prompt in output: %q", out)
	}
	if !strings.Contains(out, "my-response") {
		t.Errorf("expected response in output: %q", out)
	}
}

func TestTextWriter_Error(t *testing.T) {
	var buf bytes.Buffer
	w := NewTextWriter(&buf)
	if err := w.Write(makeErrResult(3, "q", fmt.Errorf("fail"))); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("expected ERROR in output: %q", buf.String())
	}
}
