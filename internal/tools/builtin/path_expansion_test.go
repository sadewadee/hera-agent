package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pathstest "github.com/sadewadee/hera/internal/paths/testutil"
)

// Each regression test below confirms a tool that previously called
// os.ReadFile/WriteFile/Stat directly now routes through paths.Normalize
// so ~, $HERA_HOME, and .hera/… prefixes all resolve correctly.

func TestFileWrite_TildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Point CWD somewhere unrelated so bare paths don't shadow HOME.
	t.Chdir(t.TempDir())

	tool := &FileWriteTool{}
	args, _ := json.Marshal(map[string]string{
		"path":    "~/tilde-test.txt",
		"content": "hello",
	})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("write returned error: %s", res.Content)
	}

	want := filepath.Join(home, "tilde-test.txt")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected file at %q, stat error: %v", want, err)
	}
}

func TestFileWrite_DotHeraRedirect(t *testing.T) {
	heraHome := pathstest.Isolate(t)

	tool := &FileWriteTool{}
	args, _ := json.Marshal(map[string]string{
		"path":    ".hera/workers/hello.py",
		"content": "print('hi')",
	})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("write returned error: %s", res.Content)
	}

	want := filepath.Join(heraHome, "workers", "hello.py")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected file at %q (HERA_HOME redirect), stat error: %v", want, err)
	}

	// Make sure we did NOT also write a literal .hera/ folder in CWD.
	if _, err := os.Stat(filepath.Join(heraHome, ".hera")); err == nil {
		t.Errorf("unexpected literal .hera/ folder created under CWD=%s", heraHome)
	}
}

func TestCSV_Generate_DotHeraRedirect(t *testing.T) {
	heraHome := pathstest.Isolate(t)

	tool := &CSVTool{}
	args, _ := json.Marshal(map[string]any{
		"action":  "generate",
		"headers": []string{"name", "age"},
		"rows":    [][]string{{"alice", "30"}},
		"output":  ".hera/data/people.csv",
	})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("generate returned error: %s", res.Content)
	}

	want := filepath.Join(heraHome, "data", "people.csv")
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("expected CSV at %q, read error: %v", want, err)
	}
	if !strings.Contains(string(data), "alice") {
		t.Errorf("CSV content missing expected row: %s", data)
	}
}

func TestFileRead_DollarHeraHomeExpansion(t *testing.T) {
	heraHome := pathstest.Isolate(t)

	// Seed a file under HERA_HOME.
	seed := filepath.Join(heraHome, "note.txt")
	if err := os.WriteFile(seed, []byte("seeded"), 0o644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	tool := &FileReadTool{}
	args, _ := json.Marshal(map[string]string{"path": "$HERA_HOME/note.txt"})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("read returned error: %s", res.Content)
	}
	if !strings.Contains(res.Content, "seeded") {
		t.Errorf("read content = %q, want to contain 'seeded'", res.Content)
	}
}
