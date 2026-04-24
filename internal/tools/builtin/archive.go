package builtin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// ArchiveTool handles archive and compression operations (zip, tar, gzip).
type ArchiveTool struct{}

type archiveArgs struct {
	Action string   `json:"action"`
	Format string   `json:"format,omitempty"`
	Output string   `json:"output,omitempty"`
	Input  string   `json:"input,omitempty"`
	Files  []string `json:"files,omitempty"`
}

func (t *ArchiveTool) Name() string { return "archive" }

func (t *ArchiveTool) Description() string {
	return "Creates and extracts archives. Supports zip, tar, and tar.gz formats."
}

func (t *ArchiveTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["create", "extract", "list"],
				"description": "Archive action."
			},
			"format": {
				"type": "string",
				"enum": ["zip", "tar", "tar.gz"],
				"description": "Archive format. Defaults to zip."
			},
			"output": {
				"type": "string",
				"description": "Output path for create, or extraction directory for extract."
			},
			"input": {
				"type": "string",
				"description": "Input archive path (for extract and list)."
			},
			"files": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Files to include in the archive (for create)."
			}
		},
		"required": ["action"]
	}`)
}

func (t *ArchiveTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a archiveArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	// Normalize all user-supplied paths so ~, $HERA_HOME, .hera/… all
	// resolve correctly regardless of CWD. Subfunctions can treat paths
	// as already-resolved.
	a.Output = paths.Normalize(a.Output)
	a.Input = paths.Normalize(a.Input)
	for i, f := range a.Files {
		a.Files[i] = paths.Normalize(f)
	}

	if a.Format == "" {
		a.Format = "zip"
	}

	switch a.Action {
	case "create":
		return archiveCreate(a)
	case "extract":
		return archiveExtract(a)
	case "list":
		return archiveList(a)
	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

func archiveCreate(a archiveArgs) (*tools.Result, error) {
	if len(a.Files) == 0 {
		return &tools.Result{Content: "files list is required for create", IsError: true}, nil
	}
	if a.Output == "" {
		return &tools.Result{Content: "output path is required for create", IsError: true}, nil
	}

	switch a.Format {
	case "zip":
		return createZip(a.Output, a.Files)
	case "tar", "tar.gz":
		return createTar(a.Output, a.Files, a.Format == "tar.gz")
	default:
		return &tools.Result{Content: "unsupported format: " + a.Format, IsError: true}, nil
	}
}

func createZip(output string, files []string) (*tools.Result, error) {
	f, err := os.Create(output)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create output: %v", err), IsError: true}, nil
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	count := 0
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("stat %s: %v", path, err), IsError: true}, nil
		}
		if info.IsDir() {
			continue
		}
		fw, err := w.Create(filepath.Base(path))
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("create entry: %v", err), IsError: true}, nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("read %s: %v", path, err), IsError: true}, nil
		}
		if _, err := fw.Write(data); err != nil {
			return &tools.Result{Content: fmt.Sprintf("write entry: %v", err), IsError: true}, nil
		}
		count++
	}

	return &tools.Result{Content: fmt.Sprintf("Created %s with %d files", output, count)}, nil
}

func createTar(output string, files []string, useGzip bool) (*tools.Result, error) {
	f, err := os.Create(output)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create output: %v", err), IsError: true}, nil
	}
	defer f.Close()

	var w io.Writer = f
	if useGzip {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		w = gw
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	count := 0
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("stat %s: %v", path, err), IsError: true}, nil
		}
		if info.IsDir() {
			continue
		}
		hdr := &tar.Header{
			Name: filepath.Base(path),
			Size: info.Size(),
			Mode: int64(info.Mode()),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return &tools.Result{Content: fmt.Sprintf("write header: %v", err), IsError: true}, nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("read %s: %v", path, err), IsError: true}, nil
		}
		if _, err := tw.Write(data); err != nil {
			return &tools.Result{Content: fmt.Sprintf("write data: %v", err), IsError: true}, nil
		}
		count++
	}

	return &tools.Result{Content: fmt.Sprintf("Created %s with %d files", output, count)}, nil
}

func archiveExtract(a archiveArgs) (*tools.Result, error) {
	if a.Input == "" {
		return &tools.Result{Content: "input path is required for extract", IsError: true}, nil
	}
	dest := a.Output
	if dest == "" {
		dest = "."
	}

	switch {
	case a.Format == "zip" || strings.HasSuffix(a.Input, ".zip"):
		return extractZip(a.Input, dest)
	case a.Format == "tar.gz" || strings.HasSuffix(a.Input, ".tar.gz") || strings.HasSuffix(a.Input, ".tgz"):
		return extractTarGz(a.Input, dest)
	case a.Format == "tar" || strings.HasSuffix(a.Input, ".tar"):
		return extractTar(a.Input, dest)
	default:
		return &tools.Result{Content: fmt.Sprintf("unsupported format for extract: %q", a.Format), IsError: true}, nil
	}
}

func extractZip(input, dest string) (*tools.Result, error) {
	r, err := zip.OpenReader(input)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open archive: %v", err), IsError: true}, nil
	}
	defer r.Close()

	count := 0
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		// Prevent zip slip
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dest)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0o755)
			continue
		}
		os.MkdirAll(filepath.Dir(path), 0o755)
		rc, err := f.Open()
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("open entry: %v", err), IsError: true}, nil
		}
		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return &tools.Result{Content: fmt.Sprintf("create file: %v", err), IsError: true}, nil
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("write file: %v", err), IsError: true}, nil
		}
		count++
	}

	return &tools.Result{Content: fmt.Sprintf("Extracted %d files to %s", count, dest)}, nil
}

func archiveList(a archiveArgs) (*tools.Result, error) {
	if a.Input == "" {
		return &tools.Result{Content: "input path is required for list", IsError: true}, nil
	}

	switch {
	case a.Format == "zip" || strings.HasSuffix(a.Input, ".zip"):
		r, err := zip.OpenReader(a.Input)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("open archive: %v", err), IsError: true}, nil
		}
		defer r.Close()
		var sb strings.Builder
		for _, f := range r.File {
			fmt.Fprintf(&sb, "%s (%d bytes)\n", f.Name, f.UncompressedSize64)
		}
		return &tools.Result{Content: sb.String()}, nil
	case a.Format == "tar.gz" || strings.HasSuffix(a.Input, ".tar.gz") || strings.HasSuffix(a.Input, ".tgz"):
		return listTarGz(a.Input)
	case a.Format == "tar" || strings.HasSuffix(a.Input, ".tar"):
		return listTar(a.Input)
	default:
		return &tools.Result{Content: fmt.Sprintf("unsupported format for list: %q", a.Format), IsError: true}, nil
	}
}

// extractTar extracts a plain tar archive to dest.
// Zip-slip prevention: entries that resolve outside dest are skipped.
func extractTar(input, dest string) (*tools.Result, error) {
	f, err := os.Open(input)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open archive: %v", err), IsError: true}, nil
	}
	defer f.Close()
	return extractTarReader(tar.NewReader(f), dest)
}

// extractTarGz extracts a gzip-compressed tar archive to dest.
func extractTarGz(input, dest string) (*tools.Result, error) {
	f, err := os.Open(input)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open archive: %v", err), IsError: true}, nil
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open gzip stream: %v", err), IsError: true}, nil
	}
	defer gr.Close()
	return extractTarReader(tar.NewReader(gr), dest)
}

// extractTarReader reads entries from tr and writes them under dest.
func extractTarReader(tr *tar.Reader, dest string) (*tools.Result, error) {
	cleanDest := filepath.Clean(dest)
	count := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("read tar entry: %v", err), IsError: true}, nil
		}

		target := filepath.Join(dest, filepath.Clean(hdr.Name))
		// Zip-slip / path-traversal prevention.
		if !strings.HasPrefix(filepath.Clean(target), cleanDest) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return &tools.Result{Content: fmt.Sprintf("mkdir: %v", err), IsError: true}, nil
			}
		default:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return &tools.Result{Content: fmt.Sprintf("mkdir: %v", err), IsError: true}, nil
			}
			out, err := os.Create(target)
			if err != nil {
				return &tools.Result{Content: fmt.Sprintf("create file: %v", err), IsError: true}, nil
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return &tools.Result{Content: fmt.Sprintf("write file: %v", err), IsError: true}, nil
			}
			out.Close()
			count++
		}
	}
	return &tools.Result{Content: fmt.Sprintf("Extracted %d files to %s", count, dest)}, nil
}

// listTar lists the contents of a plain tar archive.
func listTar(input string) (*tools.Result, error) {
	f, err := os.Open(input)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open archive: %v", err), IsError: true}, nil
	}
	defer f.Close()
	return listTarReader(tar.NewReader(f))
}

// listTarGz lists the contents of a gzip-compressed tar archive.
func listTarGz(input string) (*tools.Result, error) {
	f, err := os.Open(input)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open archive: %v", err), IsError: true}, nil
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open gzip stream: %v", err), IsError: true}, nil
	}
	defer gr.Close()
	return listTarReader(tar.NewReader(gr))
}

// listTarReader iterates all headers in tr and returns a formatted listing.
func listTarReader(tr *tar.Reader) (*tools.Result, error) {
	var sb strings.Builder
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("read tar entry: %v", err), IsError: true}, nil
		}
		fmt.Fprintf(&sb, "%s (%d bytes)\n", hdr.Name, hdr.Size)
	}
	return &tools.Result{Content: sb.String()}, nil
}

// RegisterArchive registers the archive tool with the given registry.
func RegisterArchive(registry *tools.Registry) {
	registry.Register(&ArchiveTool{})
}
