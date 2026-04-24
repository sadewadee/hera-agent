package batch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource_Prompts(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "prompts.txt")
	content := "# comment\n\nhello world\n  spaces  \n# another comment\nfoo bar\n"
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	src := NewFileSource(f)
	prompts, err := src.Prompts()
	if err != nil {
		t.Fatalf("Prompts() error: %v", err)
	}

	want := []string{"hello world", "spaces", "foo bar"}
	if len(prompts) != len(want) {
		t.Fatalf("got %d prompts, want %d: %v", len(prompts), len(want), prompts)
	}
	for i, p := range prompts {
		if p != want[i] {
			t.Errorf("[%d] got %q, want %q", i, p, want[i])
		}
	}
}

func TestFileSource_NotFound(t *testing.T) {
	src := NewFileSource("/nonexistent/path/prompts.txt")
	_, err := src.Prompts()
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestSliceSource_Prompts(t *testing.T) {
	input := []string{"a", "b", "c"}
	src := NewSliceSource(input)
	got, err := src.Prompts()
	if err != nil {
		t.Fatalf("Prompts() error: %v", err)
	}
	if len(got) != len(input) {
		t.Fatalf("got %d, want %d", len(got), len(input))
	}
	for i, p := range got {
		if p != input[i] {
			t.Errorf("[%d] got %q, want %q", i, p, input[i])
		}
	}
}

func TestSliceSource_Empty(t *testing.T) {
	src := NewSliceSource(nil)
	got, err := src.Prompts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}
