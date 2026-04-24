package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryExtTool_Name(t *testing.T) {
	tool := &BinaryExtTool{}
	assert.Equal(t, "binary_ext", tool.Name())
}

func TestBinaryExtTool_InvalidArgs(t *testing.T) {
	tool := &BinaryExtTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBinaryExtTool_FileNotFound(t *testing.T) {
	tool := &BinaryExtTool{}
	args, _ := json.Marshal(binaryExtArgs{FilePath: "/nonexistent/file.xyz"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBinaryExtTool_TextFile(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.go")
	os.WriteFile(fpath, []byte("package main"), 0644)

	tool := &BinaryExtTool{}
	args, _ := json.Marshal(binaryExtArgs{FilePath: fpath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, ".go")
	assert.Contains(t, result.Content, "Binary: false")
}

func TestBinaryExtTool_BinaryFile_PNG(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "image.png")
	os.WriteFile(fpath, []byte{0x89, 0x50, 0x4E, 0x47}, 0644)

	tool := &BinaryExtTool{}
	args, _ := json.Marshal(binaryExtArgs{FilePath: fpath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, ".png")
	assert.Contains(t, result.Content, "Binary: true")
}

func TestBinaryExtTool_BinaryFile_PDF(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "doc.pdf")
	os.WriteFile(fpath, []byte("%PDF-1.4"), 0644)

	tool := &BinaryExtTool{}
	args, _ := json.Marshal(binaryExtArgs{FilePath: fpath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Binary: true")
}

func TestBinaryExtTool_BinaryFile_ZIP(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "archive.zip")
	os.WriteFile(fpath, []byte("PK\x03\x04"), 0644)

	tool := &BinaryExtTool{}
	args, _ := json.Marshal(binaryExtArgs{FilePath: fpath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Binary: true")
}

func TestBinaryExtTool_ShowsFileSize(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "data.txt")
	os.WriteFile(fpath, []byte("hello world"), 0644)

	tool := &BinaryExtTool{}
	args, _ := json.Marshal(binaryExtArgs{FilePath: fpath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "11 bytes")
}
