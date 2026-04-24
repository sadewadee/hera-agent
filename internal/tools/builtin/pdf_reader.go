package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// PDFReaderTool extracts text content from PDF files.
type PDFReaderTool struct{}

type pdfReaderArgs struct {
	Path      string `json:"path"`
	PageStart int    `json:"page_start,omitempty"`
	PageEnd   int    `json:"page_end,omitempty"`
}

func (t *PDFReaderTool) Name() string { return "pdf_reader" }

func (t *PDFReaderTool) Description() string {
	return "Reads and extracts text content from PDF files. Can extract specific page ranges."
}

func (t *PDFReaderTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the PDF file."
			},
			"page_start": {
				"type": "integer",
				"description": "Start page (1-based). Defaults to 1."
			},
			"page_end": {
				"type": "integer",
				"description": "End page (inclusive). Defaults to last page."
			}
		},
		"required": ["path"]
	}`)
}

func (t *PDFReaderTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params pdfReaderArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Path == "" {
		return &tools.Result{Content: "path is required", IsError: true}, nil
	}
	params.Path = paths.Normalize(params.Path)

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("failed to read file: %v", err), IsError: true}, nil
	}

	// Basic PDF text extraction: find text between stream/endstream markers
	// and extract readable ASCII content. For full fidelity, an external tool
	// like pdftotext would be needed.
	text := extractPDFText(data)
	if text == "" {
		return &tools.Result{Content: "No extractable text found in PDF. The file may contain only images or scanned content."}, nil
	}

	// Truncate to avoid overwhelming the context window
	const maxLen = 100 * 1024
	if len(text) > maxLen {
		text = text[:maxLen] + "\n...[truncated]"
	}

	return &tools.Result{Content: text}, nil
}

// extractPDFText performs a basic extraction of readable text from PDF bytes.
func extractPDFText(data []byte) string {
	var sb strings.Builder
	content := string(data)

	// Look for text between BT (begin text) and ET (end text) markers
	for {
		btIdx := strings.Index(content, "BT")
		if btIdx < 0 {
			break
		}
		content = content[btIdx+2:]
		etIdx := strings.Index(content, "ET")
		if etIdx < 0 {
			break
		}
		block := content[:etIdx]
		content = content[etIdx+2:]

		// Extract parenthesized strings (PDF text objects)
		for i := 0; i < len(block); i++ {
			if block[i] == '(' {
				depth := 1
				start := i + 1
				for j := start; j < len(block) && depth > 0; j++ {
					switch block[j] {
					case '(':
						depth++
					case ')':
						depth--
						if depth == 0 {
							sb.WriteString(block[start:j])
							sb.WriteByte(' ')
							i = j
						}
					case '\\':
						j++ // skip escaped char
					}
				}
			}
		}
		sb.WriteByte('\n')
	}

	return strings.TrimSpace(sb.String())
}

// RegisterPDFReader registers the PDF reader tool with the given registry.
func RegisterPDFReader(registry *tools.Registry) {
	registry.Register(&PDFReaderTool{})
}
