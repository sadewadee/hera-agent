package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReadTool_Name(t *testing.T) {
	tool := &FileReadTool{}
	if got := tool.Name(); got != "file_read" {
		t.Errorf("Name() = %q, want %q", got, "file_read")
	}
}

func TestFileReadTool_Execute_ReadFile(t *testing.T) {
	// Create a temporary file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello, world"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &FileReadTool{}
	args, _ := json.Marshal(fileReadArgs{Path: path})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if result.Content != content {
		t.Errorf("Execute() content = %q, want %q", result.Content, content)
	}
}

func TestFileReadTool_Execute_FileNotFound(t *testing.T) {
	tool := &FileReadTool{}
	args, _ := json.Marshal(fileReadArgs{Path: "/nonexistent/file.txt"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for nonexistent file")
	}
}

func TestFileReadTool_Execute_EmptyPath(t *testing.T) {
	tool := &FileReadTool{}
	args, _ := json.Marshal(fileReadArgs{Path: ""})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for empty path")
	}
}

func TestFileReadTool_Execute_ProtectedPath(t *testing.T) {
	dir := t.TempDir()
	protectedDir := filepath.Join(dir, "protected")
	if err := os.MkdirAll(protectedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	protectedFile := filepath.Join(protectedDir, "secret.txt")
	if err := os.WriteFile(protectedFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &FileReadTool{protectedPaths: []string{protectedDir}}
	args, _ := json.Marshal(fileReadArgs{Path: protectedFile})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for protected path")
	}
	if !strings.Contains(result.Content, "access denied") {
		t.Errorf("error should mention access denied, got: %q", result.Content)
	}
}

func TestFileReadTool_Execute_TruncatesLargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	// Write a file larger than 100KB
	data := strings.Repeat("x", 120*1024)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &FileReadTool{}
	args, _ := json.Marshal(fileReadArgs{Path: path})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "truncated") {
		t.Error("Execute() should indicate truncation for large files")
	}
}

func TestFileWriteTool_Name(t *testing.T) {
	tool := &FileWriteTool{}
	if got := tool.Name(); got != "file_write" {
		t.Errorf("Name() = %q, want %q", got, "file_write")
	}
}

func TestFileWriteTool_Execute_WriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")
	content := "hello, world"

	tool := &FileWriteTool{}
	args, _ := json.Marshal(fileWriteArgs{Path: path, Content: content})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}

	// Verify file was written
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestFileWriteTool_Execute_Append(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "append.txt")
	if err := os.WriteFile(path, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &FileWriteTool{}
	args, _ := json.Marshal(fileWriteArgs{Path: path, Content: "second\n", Append: true})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "first\nsecond\n" {
		t.Errorf("file content = %q, want %q", string(data), "first\nsecond\n")
	}
}

func TestFileWriteTool_Execute_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "file.txt")

	tool := &FileWriteTool{}
	args, _ := json.Marshal(fileWriteArgs{Path: path, Content: "nested"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "nested" {
		t.Errorf("file content = %q, want %q", string(data), "nested")
	}
}

func TestFileWriteTool_Execute_ProtectedPath(t *testing.T) {
	dir := t.TempDir()
	protectedDir := filepath.Join(dir, "protected")
	if err := os.MkdirAll(protectedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tool := &FileWriteTool{protectedPaths: []string{protectedDir}}
	args, _ := json.Marshal(fileWriteArgs{Path: filepath.Join(protectedDir, "file.txt"), Content: "data"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for protected path")
	}
}

func TestExpandAndAbs(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tilde expansion",
			path: "~/test",
			want: filepath.Join(home, "test"),
		},
		{
			name: "absolute path unchanged",
			path: "/tmp/test",
			want: "/tmp/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandAndAbs(tt.path)
			if err != nil {
				t.Fatalf("expandAndAbs() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("expandAndAbs(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsProtected(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		protectedPaths []string
		want           bool
	}{
		{
			name:           "exact match",
			path:           "/home/user/.ssh",
			protectedPaths: []string{"/home/user/.ssh"},
			want:           true,
		},
		{
			name:           "inside protected",
			path:           "/home/user/.ssh/id_rsa",
			protectedPaths: []string{"/home/user/.ssh"},
			want:           true,
		},
		{
			name:           "outside protected",
			path:           "/home/user/documents/file.txt",
			protectedPaths: []string{"/home/user/.ssh"},
			want:           false,
		},
		{
			name:           "empty protected list",
			path:           "/any/path",
			protectedPaths: nil,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isProtected(tt.path, tt.protectedPaths)
			if got != tt.want {
				t.Errorf("isProtected(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
