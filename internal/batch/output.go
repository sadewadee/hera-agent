package batch

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// OutputWriter consumes PromptResults.
type OutputWriter interface {
	Write(r PromptResult) error
	Close() error
}

// JSONLRecord is the JSON structure written per line by JSONLWriter.
type JSONLRecord struct {
	Index      int    `json:"index"`
	Prompt     string `json:"prompt"`
	Response   string `json:"response"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

// JSONLWriter writes one JSON object per line (JSONL / NDJSON format).
type JSONLWriter struct {
	mu sync.Mutex
	w  io.Writer
	f  *os.File // non-nil when we opened the file ourselves
}

// NewJSONLWriter creates a JSONLWriter writing to w.
func NewJSONLWriter(w io.Writer) *JSONLWriter {
	return &JSONLWriter{w: w}
}

// NewJSONLFileWriter opens path for appending and returns a JSONLWriter.
// Close() must be called to flush and close the file.
func NewJSONLFileWriter(path string) (*JSONLWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open output file %q: %w", path, err)
	}
	return &JSONLWriter{w: f, f: f}, nil
}

// Write appends a JSON line for r.
func (w *JSONLWriter) Write(r PromptResult) error {
	rec := JSONLRecord{
		Index:      r.Index,
		Prompt:     r.Prompt,
		Response:   r.Response,
		DurationMs: r.Duration.Milliseconds(),
	}
	if r.Err != nil {
		rec.Error = r.Err.Error()
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = fmt.Fprintf(w.w, "%s\n", data)
	return err
}

// Close closes the underlying file if JSONLWriter owns it.
func (w *JSONLWriter) Close() error {
	if w.f != nil {
		return w.f.Close()
	}
	return nil
}

// CSVWriter writes results as CSV rows: index,prompt,response,error,duration_ms
type CSVWriter struct {
	mu     sync.Mutex
	cw     *csv.Writer
	f      *os.File
	header bool
}

// NewCSVWriter creates a CSVWriter writing to w.
func NewCSVWriter(w io.Writer) *CSVWriter {
	return &CSVWriter{cw: csv.NewWriter(w)}
}

// NewCSVFileWriter opens path and returns a CSVWriter.
func NewCSVFileWriter(path string) (*CSVWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open csv file %q: %w", path, err)
	}
	return &CSVWriter{cw: csv.NewWriter(f), f: f}, nil
}

// Write appends a CSV row for r.
func (w *CSVWriter) Write(r PromptResult) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.header {
		if err := w.cw.Write([]string{"index", "prompt", "response", "error", "duration_ms"}); err != nil {
			return err
		}
		w.header = true
	}

	errStr := ""
	if r.Err != nil {
		errStr = r.Err.Error()
	}
	row := []string{
		fmt.Sprintf("%d", r.Index),
		r.Prompt,
		r.Response,
		errStr,
		fmt.Sprintf("%d", r.Duration.Milliseconds()),
	}
	if err := w.cw.Write(row); err != nil {
		return err
	}
	w.cw.Flush()
	return w.cw.Error()
}

// Close closes the underlying file if CSVWriter owns it.
func (w *CSVWriter) Close() error {
	w.cw.Flush()
	if w.f != nil {
		return w.f.Close()
	}
	return nil
}

// TextWriter writes human-readable text: "--- [N] prompt\nresponse\n".
type TextWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// NewTextWriter creates a TextWriter writing to w.
func NewTextWriter(w io.Writer) *TextWriter {
	return &TextWriter{w: w}
}

// Write appends a text block for r.
func (w *TextWriter) Write(r PromptResult) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	ts := time.Now().Format(time.RFC3339)
	if r.Err != nil {
		_, err := fmt.Fprintf(w.w, "--- [%d] %s\nERROR: %v\n\n", r.Index, ts, r.Err)
		return err
	}
	_, err := fmt.Fprintf(w.w, "--- [%d] %s\nPROMPT: %s\nRESPONSE:\n%s\n\n", r.Index, ts, r.Prompt, r.Response)
	return err
}

// Close is a no-op for TextWriter (it does not own the underlying writer).
func (w *TextWriter) Close() error { return nil }
