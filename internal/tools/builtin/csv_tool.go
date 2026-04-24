package builtin

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// CSVTool provides CSV parsing and generation capabilities.
type CSVTool struct{}

type csvToolArgs struct {
	Action    string     `json:"action"`
	Path      string     `json:"path,omitempty"`
	Data      string     `json:"data,omitempty"`
	Delimiter string     `json:"delimiter,omitempty"`
	Headers   []string   `json:"headers,omitempty"`
	Rows      [][]string `json:"rows,omitempty"`
	Output    string     `json:"output,omitempty"`
}

func (t *CSVTool) Name() string { return "csv" }

func (t *CSVTool) Description() string {
	return "CSV operations: read, parse, convert to JSON, generate CSV from data."
}

func (t *CSVTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["read", "parse", "to_json", "generate"],
				"description": "CSV action: read (from file), parse (from string), to_json (convert to JSON), generate (create CSV)."
			},
			"path": {"type": "string", "description": "File path (for read action)."},
			"data": {"type": "string", "description": "CSV data as string (for parse/to_json)."},
			"delimiter": {"type": "string", "description": "Column delimiter. Defaults to comma."},
			"headers": {"type": "array", "items": {"type": "string"}, "description": "Column headers (for generate)."},
			"rows": {"type": "array", "items": {"type": "array", "items": {"type": "string"}}, "description": "Data rows (for generate)."},
			"output": {"type": "string", "description": "Output file path (for generate)."}
		},
		"required": ["action"]
	}`)
}

func (t *CSVTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a csvToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	// Normalize any user-supplied paths so ~, $HERA_HOME, and .hera/…
	// work regardless of CWD.
	a.Path = paths.Normalize(a.Path)
	a.Output = paths.Normalize(a.Output)

	delim := ','
	if a.Delimiter != "" {
		delim = rune(a.Delimiter[0])
	}

	switch a.Action {
	case "read":
		if a.Path == "" {
			return &tools.Result{Content: "path is required for read", IsError: true}, nil
		}
		data, err := os.ReadFile(a.Path)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("read file: %v", err), IsError: true}, nil
		}
		return csvParse(string(data), delim)

	case "parse":
		if a.Data == "" {
			return &tools.Result{Content: "data is required for parse", IsError: true}, nil
		}
		return csvParse(a.Data, delim)

	case "to_json":
		data := a.Data
		if data == "" && a.Path != "" {
			content, err := os.ReadFile(a.Path)
			if err != nil {
				return &tools.Result{Content: fmt.Sprintf("read file: %v", err), IsError: true}, nil
			}
			data = string(content)
		}
		if data == "" {
			return &tools.Result{Content: "data or path is required", IsError: true}, nil
		}
		return csvToJSON(data, delim)

	case "generate":
		return csvGenerate(a.Headers, a.Rows, a.Output, delim)

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

func csvParse(data string, delim rune) (*tools.Result, error) {
	r := csv.NewReader(strings.NewReader(data))
	r.Comma = delim
	r.LazyQuotes = true

	records, err := r.ReadAll()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("parse error: %v", err), IsError: true}, nil
	}

	var sb strings.Builder
	for i, row := range records {
		if i > 1000 {
			sb.WriteString(fmt.Sprintf("\n...[truncated at 1000 rows, total: %d]", len(records)))
			break
		}
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteString("\n")
	}

	return &tools.Result{Content: fmt.Sprintf("%d rows, %d columns\n\n%s", len(records), len(records[0]), strings.TrimSpace(sb.String()))}, nil
}

func csvToJSON(data string, delim rune) (*tools.Result, error) {
	r := csv.NewReader(strings.NewReader(data))
	r.Comma = delim
	r.LazyQuotes = true

	records, err := r.ReadAll()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("parse error: %v", err), IsError: true}, nil
	}

	if len(records) < 2 {
		return &tools.Result{Content: "CSV must have at least a header row and one data row", IsError: true}, nil
	}

	headers := records[0]
	var result []map[string]string
	for _, row := range records[1:] {
		obj := make(map[string]string)
		for i, h := range headers {
			if i < len(row) {
				obj[h] = row[i]
			}
		}
		result = append(result, obj)
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("marshal error: %v", err), IsError: true}, nil
	}

	return &tools.Result{Content: string(out)}, nil
}

func csvGenerate(headers []string, rows [][]string, output string, delim rune) (*tools.Result, error) {
	if len(headers) == 0 && len(rows) == 0 {
		return &tools.Result{Content: "headers or rows are required", IsError: true}, nil
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	w.Comma = delim

	if len(headers) > 0 {
		if err := w.Write(headers); err != nil {
			return &tools.Result{Content: fmt.Sprintf("write headers: %v", err), IsError: true}, nil
		}
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return &tools.Result{Content: fmt.Sprintf("write row: %v", err), IsError: true}, nil
		}
	}
	w.Flush()

	csvData := sb.String()
	if output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return &tools.Result{Content: fmt.Sprintf("create output dir: %v", err), IsError: true}, nil
		}
		if err := os.WriteFile(output, []byte(csvData), 0o644); err != nil {
			return &tools.Result{Content: fmt.Sprintf("write file: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("CSV written to %s (%d rows)", output, len(rows))}, nil
	}

	return &tools.Result{Content: csvData}, nil
}

// RegisterCSV registers the CSV tool with the given registry.
func RegisterCSV(registry *tools.Registry) {
	registry.Register(&CSVTool{})
}
