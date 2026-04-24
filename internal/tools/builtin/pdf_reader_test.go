package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDFReaderTool_Name(t *testing.T) {
	tool := &PDFReaderTool{}
	assert.Equal(t, "pdf_reader", tool.Name())
}

func TestPDFReaderTool_Description(t *testing.T) {
	tool := &PDFReaderTool{}
	assert.Contains(t, tool.Description(), "PDF")
}

func TestPDFReaderTool_InvalidArgs(t *testing.T) {
	tool := &PDFReaderTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestPDFReaderTool_EmptyPath(t *testing.T) {
	tool := &PDFReaderTool{}
	args, _ := json.Marshal(pdfReaderArgs{Path: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "path is required")
}

func TestPDFReaderTool_FileNotFound(t *testing.T) {
	tool := &PDFReaderTool{}
	args, _ := json.Marshal(pdfReaderArgs{Path: "/nonexistent/file.pdf"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "failed to read file")
}

func TestPDFReaderTool_ReadPDFWithText(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "test.pdf")
	// Create a minimal PDF-like file with BT/ET text markers
	content := "%PDF-1.4\nBT\n(Hello World) Tj\nET\n%%EOF"
	require.NoError(t, os.WriteFile(pdfPath, []byte(content), 0o644))

	tool := &PDFReaderTool{}
	args, _ := json.Marshal(pdfReaderArgs{Path: pdfPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Hello World")
}

func TestPDFReaderTool_NoText(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "empty.pdf")
	require.NoError(t, os.WriteFile(pdfPath, []byte("%PDF-1.4\n%%EOF"), 0o644))

	tool := &PDFReaderTool{}
	args, _ := json.Marshal(pdfReaderArgs{Path: pdfPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "No extractable text")
}

func TestExtractPDFText_Empty(t *testing.T) {
	result := extractPDFText([]byte("no markers here"))
	assert.Empty(t, result)
}

func TestExtractPDFText_NestedParens(t *testing.T) {
	data := []byte("BT\n(Hello (nested) world) Tj\nET")
	result := extractPDFText(data)
	assert.Contains(t, result, "Hello")
}

func TestRegisterPDFReader(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterPDFReader(registry)
	_, ok := registry.Get("pdf_reader")
	assert.True(t, ok)
}
