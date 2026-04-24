package builtin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveTool_Name(t *testing.T) {
	tool := &ArchiveTool{}
	assert.Equal(t, "archive", tool.Name())
}

func TestArchiveTool_Description(t *testing.T) {
	tool := &ArchiveTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestArchiveTool_Parameters(t *testing.T) {
	tool := &ArchiveTool{}
	var schema map[string]interface{}
	err := json.Unmarshal(tool.Parameters(), &schema)
	require.NoError(t, err)
	assert.Equal(t, "object", schema["type"])
}

func TestArchiveTool_InvalidJSON(t *testing.T) {
	tool := &ArchiveTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid arguments")
}

func TestArchiveTool_UnknownAction(t *testing.T) {
	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "compress"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestArchiveTool_CreateZip(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "a.txt")
	file2 := filepath.Join(dir, "b.txt")
	os.WriteFile(file1, []byte("hello"), 0o644)
	os.WriteFile(file2, []byte("world"), 0o644)

	output := filepath.Join(dir, "test.zip")
	args, _ := json.Marshal(archiveArgs{
		Action: "create",
		Format: "zip",
		Output: output,
		Files:  []string{file1, file2},
	})
	tool := &ArchiveTool{}
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "unexpected error: %s", result.Content)
	assert.Contains(t, result.Content, "2 files")

	_, statErr := os.Stat(output)
	assert.NoError(t, statErr)
}

func TestArchiveTool_CreateTar(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "x.txt")
	os.WriteFile(file1, []byte("tar content"), 0o644)

	output := filepath.Join(dir, "test.tar")
	args, _ := json.Marshal(archiveArgs{
		Action: "create",
		Format: "tar",
		Output: output,
		Files:  []string{file1},
	})
	tool := &ArchiveTool{}
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "unexpected error: %s", result.Content)
	assert.Contains(t, result.Content, "1 files")
}

func TestArchiveTool_CreateTarGz(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "gz.txt")
	os.WriteFile(file1, []byte("gzipped content"), 0o644)

	output := filepath.Join(dir, "test.tar.gz")
	args, _ := json.Marshal(archiveArgs{
		Action: "create",
		Format: "tar.gz",
		Output: output,
		Files:  []string{file1},
	})
	tool := &ArchiveTool{}
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "unexpected error: %s", result.Content)
	assert.Contains(t, result.Content, "1 files")
}

func TestArchiveTool_CreateMissingFiles(t *testing.T) {
	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "create", Format: "zip", Output: "/tmp/x.zip"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "files list is required")
}

func TestArchiveTool_CreateMissingOutput(t *testing.T) {
	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "create", Format: "zip", Files: []string{"a.txt"}})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "output path is required")
}

func TestArchiveTool_ExtractZip(t *testing.T) {
	dir := t.TempDir()

	zipPath := filepath.Join(dir, "extract_me.zip")
	zf, err := os.Create(zipPath)
	require.NoError(t, err)
	w := zip.NewWriter(zf)
	fw, _ := w.Create("extracted.txt")
	fw.Write([]byte("extracted content"))
	w.Close()
	zf.Close()

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0o755)

	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{
		Action: "extract",
		Format: "zip",
		Input:  zipPath,
		Output: extractDir,
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "unexpected error: %s", result.Content)
	assert.Contains(t, result.Content, "Extracted")

	data, readErr := os.ReadFile(filepath.Join(extractDir, "extracted.txt"))
	assert.NoError(t, readErr)
	assert.Equal(t, "extracted content", string(data))
}

func TestArchiveTool_ExtractMissingInput(t *testing.T) {
	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "extract", Format: "zip"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "input path is required")
}

func TestArchiveTool_ListZip(t *testing.T) {
	dir := t.TempDir()

	zipPath := filepath.Join(dir, "list_me.zip")
	zf, err := os.Create(zipPath)
	require.NoError(t, err)
	w := zip.NewWriter(zf)
	fw, _ := w.Create("file1.txt")
	fw.Write([]byte("111"))
	fw2, _ := w.Create("file2.txt")
	fw2.Write([]byte("222222"))
	w.Close()
	zf.Close()

	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "list", Format: "zip", Input: zipPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "unexpected error: %s", result.Content)
	assert.Contains(t, result.Content, "file1.txt")
	assert.Contains(t, result.Content, "file2.txt")
}

func TestArchiveTool_ListMissingInput(t *testing.T) {
	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "list"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "input path is required")
}

func TestArchiveTool_DefaultFormat(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "default.txt")
	os.WriteFile(file1, []byte("default format"), 0o644)
	output := filepath.Join(dir, "default.zip")

	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{
		Action: "create",
		Output: output,
		Files:  []string{file1},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "unexpected error: %s", result.Content)
	assert.Contains(t, result.Content, "1 files")
}

func TestArchiveTool_CreateSkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0o755)
	file1 := filepath.Join(dir, "keep.txt")
	os.WriteFile(file1, []byte("keep"), 0o644)

	output := filepath.Join(dir, "nodirs.zip")
	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{
		Action: "create",
		Format: "zip",
		Output: output,
		Files:  []string{subdir, file1},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "1 files")
}

func TestArchiveTool_UnsupportedFormat(t *testing.T) {
	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{
		Action: "create",
		Format: "7z",
		Output: "/tmp/test.7z",
		Files:  []string{"/tmp/x"},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unsupported format")
}

func TestArchiveTool_ZipSlipPrevention(t *testing.T) {
	dir := t.TempDir()

	zipPath := filepath.Join(dir, "malicious.zip")
	zf, err := os.Create(zipPath)
	require.NoError(t, err)
	w := zip.NewWriter(zf)
	fw, _ := w.Create("../../../etc/passwd")
	fw.Write([]byte("malicious"))
	w.Close()
	zf.Close()

	extractDir := filepath.Join(dir, "safe")
	os.MkdirAll(extractDir, 0o755)

	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{
		Action: "extract",
		Format: "zip",
		Input:  zipPath,
		Output: extractDir,
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Extracted 0 files")

	_, statErr := os.Stat(filepath.Join(dir, "etc", "passwd"))
	assert.True(t, os.IsNotExist(statErr), "zip-slip: file should not have been created outside dest")
}

func TestRegisterArchive(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterArchive(registry)
	_, ok := registry.Get("archive")
	assert.True(t, ok)
}

// --- tar extract and list (TDD: these must fail before implementation) ---

func makeTarGz(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	outPath := filepath.Join(dir, "test.tar.gz")
	f, err := os.Create(outPath)
	require.NoError(t, err)
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())
	return outPath
}

func TestArchiveTool_ExtractTarGz_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Build a tar.gz with two text files using the archive tool itself.
	src1 := filepath.Join(dir, "alpha.txt")
	src2 := filepath.Join(dir, "beta.txt")
	require.NoError(t, os.WriteFile(src1, []byte("hello alpha"), 0o644))
	require.NoError(t, os.WriteFile(src2, []byte("hello beta"), 0o644))

	archivePath := filepath.Join(dir, "round.tar.gz")
	tool := &ArchiveTool{}

	createArgs, _ := json.Marshal(archiveArgs{
		Action: "create",
		Format: "tar.gz",
		Output: archivePath,
		Files:  []string{src1, src2},
	})
	createResult, err := tool.Execute(context.Background(), createArgs)
	require.NoError(t, err)
	require.False(t, createResult.IsError, "create: %s", createResult.Content)

	// Extract into a fresh directory.
	extractDir := filepath.Join(dir, "extracted")
	require.NoError(t, os.MkdirAll(extractDir, 0o755))

	extractArgs, _ := json.Marshal(archiveArgs{
		Action: "extract",
		Format: "tar.gz",
		Input:  archivePath,
		Output: extractDir,
	})
	extractResult, err := tool.Execute(context.Background(), extractArgs)
	require.NoError(t, err)
	assert.False(t, extractResult.IsError, "extract: %s", extractResult.Content)
	assert.Contains(t, extractResult.Content, "Extracted")

	// Verify content is intact.
	data, err := os.ReadFile(filepath.Join(extractDir, "alpha.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello alpha", string(data))
}

func TestArchiveTool_ExtractTar_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "gamma.txt")
	require.NoError(t, os.WriteFile(src, []byte("gamma content"), 0o644))

	archivePath := filepath.Join(dir, "round.tar")
	tool := &ArchiveTool{}

	createArgs, _ := json.Marshal(archiveArgs{
		Action: "create",
		Format: "tar",
		Output: archivePath,
		Files:  []string{src},
	})
	createResult, err := tool.Execute(context.Background(), createArgs)
	require.NoError(t, err)
	require.False(t, createResult.IsError, "create: %s", createResult.Content)

	extractDir := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(extractDir, 0o755))

	extractArgs, _ := json.Marshal(archiveArgs{
		Action: "extract",
		Format: "tar",
		Input:  archivePath,
		Output: extractDir,
	})
	extractResult, err := tool.Execute(context.Background(), extractArgs)
	require.NoError(t, err)
	assert.False(t, extractResult.IsError, "extract: %s", extractResult.Content)
	assert.Contains(t, extractResult.Content, "Extracted")

	data, err := os.ReadFile(filepath.Join(extractDir, "gamma.txt"))
	require.NoError(t, err)
	assert.Equal(t, "gamma content", string(data))
}

func TestArchiveTool_ListTarGz(t *testing.T) {
	dir := t.TempDir()
	archivePath := makeTarGz(t, dir, map[string]string{
		"file1.txt": "aaa",
		"file2.txt": "bbbbbb",
	})

	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "list", Format: "tar.gz", Input: archivePath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "list tar.gz: %s", result.Content)
	assert.Contains(t, result.Content, "file1.txt")
	assert.Contains(t, result.Content, "file2.txt")
}

func TestArchiveTool_ListTar(t *testing.T) {
	dir := t.TempDir()

	// Create a plain tar (no gzip) using tar.Writer directly.
	tarPath := filepath.Join(dir, "plain.tar")
	f, err := os.Create(tarPath)
	require.NoError(t, err)
	tw := tar.NewWriter(f)
	content := "plain tar content"
	hdr := &tar.Header{Name: "plain.txt", Mode: 0o644, Size: int64(len(content))}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, f.Close())

	tool := &ArchiveTool{}
	args, _ := json.Marshal(archiveArgs{Action: "list", Format: "tar", Input: tarPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "list tar: %s", result.Content)
	assert.Contains(t, result.Content, "plain.txt")
}
