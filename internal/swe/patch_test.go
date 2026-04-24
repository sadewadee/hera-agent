package swe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHunks_SimpleAddition(t *testing.T) {
	diff := `--- a/foo.go
+++ b/foo.go
@@ -1,2 +1,3 @@
 line1
+new line
 line2
`
	hunks, err := ParseHunks(diff)
	if err != nil {
		t.Fatalf("ParseHunks error: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	h := hunks[0]
	if h.OldStart != 1 || h.OldCount != 2 {
		t.Errorf("old range = (%d,%d), want (1,2)", h.OldStart, h.OldCount)
	}
	if h.NewStart != 1 || h.NewCount != 3 {
		t.Errorf("new range = (%d,%d), want (1,3)", h.NewStart, h.NewCount)
	}
	if len(h.Lines) != 3 {
		t.Fatalf("expected 3 hunk lines, got %d", len(h.Lines))
	}
	if h.Lines[0].Op != OpContext || h.Lines[0].Text != "line1" {
		t.Errorf("line[0] = (%v,%q), want (OpContext,\"line1\")", h.Lines[0].Op, h.Lines[0].Text)
	}
	if h.Lines[1].Op != OpAdd || h.Lines[1].Text != "new line" {
		t.Errorf("line[1] = (%v,%q), want (OpAdd,\"new line\")", h.Lines[1].Op, h.Lines[1].Text)
	}
	if h.Lines[2].Op != OpContext || h.Lines[2].Text != "line2" {
		t.Errorf("line[2] = (%v,%q), want (OpContext,\"line2\")", h.Lines[2].Op, h.Lines[2].Text)
	}
}

func TestParseHunks_MultipleHunks(t *testing.T) {
	diff := `--- a/foo.go
+++ b/foo.go
@@ -1,2 +1,3 @@
 line1
+added
 line2
@@ -10,2 +11,2 @@
 line10
-old
+new
`
	hunks, err := ParseHunks(diff)
	if err != nil {
		t.Fatalf("ParseHunks error: %v", err)
	}
	if len(hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(hunks))
	}
	if hunks[1].OldStart != 10 {
		t.Errorf("hunk[1].OldStart = %d, want 10", hunks[1].OldStart)
	}
}

func TestParseHunks_Deletion(t *testing.T) {
	diff := `@@ -1,3 +1,2 @@
 keep1
-remove
 keep2
`
	hunks, err := ParseHunks(diff)
	if err != nil {
		t.Fatalf("ParseHunks error: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	lines := hunks[0].Lines
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1].Op != OpDelete {
		t.Errorf("line[1].Op = %v, want OpDelete", lines[1].Op)
	}
}

func TestApplyUnifiedDiff_AddLine(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,3 @@
 line1
+middle
 line2
`
	if err := ApplyUnifiedDiff(f, diff); err != nil {
		t.Fatalf("ApplyUnifiedDiff error: %v", err)
	}
	got, _ := os.ReadFile(f)
	// lines are joined with \n; trailing \n comes from the original split
	want := "line1\nmiddle\nline2\n"
	if string(got) != want {
		t.Errorf("file content = %q, want %q", string(got), want)
	}
}

func TestApplyUnifiedDiff_RemoveLine(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("keep1\nremove\nkeep2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff := `@@ -1,3 +1,2 @@
 keep1
-remove
 keep2
`
	if err := ApplyUnifiedDiff(f, diff); err != nil {
		t.Fatalf("ApplyUnifiedDiff error: %v", err)
	}
	got, _ := os.ReadFile(f)
	want := "keep1\nkeep2\n"
	if string(got) != want {
		t.Errorf("file content = %q, want %q", string(got), want)
	}
}

func TestApplyUnifiedDiff_MismatchError(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("actual\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff := `@@ -1,1 +1,1 @@
-expected_but_not_actual
+replacement
`
	err := ApplyUnifiedDiff(f, diff)
	if err == nil {
		t.Fatal("expected error for context mismatch, got nil")
	}
}

func TestApplyUnifiedDiff_EmptyDiff(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyUnifiedDiff(f, ""); err != nil {
		t.Fatalf("empty diff should be no-op, got error: %v", err)
	}
	got, _ := os.ReadFile(f)
	if string(got) != "content\n" {
		t.Errorf("file should be unchanged, got %q", string(got))
	}
}

func TestApplyUnifiedDiff_NewFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "newfile.txt")
	// File does not exist yet.

	diff := `--- /dev/null
+++ b/newfile.txt
@@ -0,0 +1,2 @@
+line1
+line2
`
	if err := ApplyUnifiedDiff(f, diff); err != nil {
		t.Fatalf("ApplyUnifiedDiff new file error: %v", err)
	}
	got, _ := os.ReadFile(f)
	want := "line1\nline2"
	if string(got) != want {
		t.Errorf("new file content = %q, want %q", string(got), want)
	}
}
