package builtin

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDatetimeTool_Name(t *testing.T) {
	tool := &DatetimeTool{}
	if got := tool.Name(); got != "datetime" {
		t.Errorf("Name() = %q, want %q", got, "datetime")
	}
}

func TestDatetimeTool_Description(t *testing.T) {
	tool := &DatetimeTool{}
	if got := tool.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

func TestDatetimeTool_Parameters(t *testing.T) {
	tool := &DatetimeTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("Parameters() returned invalid JSON")
	}
}

func TestDatetimeTool_Execute_DefaultFormat(t *testing.T) {
	tool := &DatetimeTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error result: %s", result.Content)
	}
	// Should parse as RFC3339
	if _, err := time.Parse(time.RFC3339, result.Content); err != nil {
		t.Errorf("Execute() default format not RFC3339: %q, error: %v", result.Content, err)
	}
}

func TestDatetimeTool_Execute_Formats(t *testing.T) {
	tool := &DatetimeTool{}

	tests := []struct {
		name   string
		format string
		check  func(t *testing.T, output string)
	}{
		{
			name:   "iso8601",
			format: "iso8601",
			check: func(t *testing.T, output string) {
				if _, err := time.Parse(time.RFC3339, output); err != nil {
					t.Errorf("iso8601 format invalid: %q", output)
				}
			},
		},
		{
			name:   "rfc3339",
			format: "rfc3339",
			check: func(t *testing.T, output string) {
				if _, err := time.Parse(time.RFC3339, output); err != nil {
					t.Errorf("rfc3339 format invalid: %q", output)
				}
			},
		},
		{
			name:   "unix",
			format: "unix",
			check: func(t *testing.T, output string) {
				if _, err := strconv.ParseInt(output, 10, 64); err != nil {
					t.Errorf("unix format not an integer: %q", output)
				}
			},
		},
		{
			name:   "date",
			format: "date",
			check: func(t *testing.T, output string) {
				if _, err := time.Parse("2006-01-02", output); err != nil {
					t.Errorf("date format invalid: %q", output)
				}
			},
		},
		{
			name:   "time",
			format: "time",
			check: func(t *testing.T, output string) {
				if _, err := time.Parse("15:04:05", output); err != nil {
					t.Errorf("time format invalid: %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(datetimeArgs{Format: tt.format})
			result, err := tool.Execute(context.Background(), args)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if result.IsError {
				t.Fatalf("Execute() returned error: %s", result.Content)
			}
			tt.check(t, result.Content)
		})
	}
}

func TestDatetimeTool_Execute_Timezone(t *testing.T) {
	tool := &DatetimeTool{}
	args, _ := json.Marshal(datetimeArgs{Format: "date", Timezone: "UTC"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	// Should be a valid date
	if _, err := time.Parse("2006-01-02", result.Content); err != nil {
		t.Errorf("timezone output not a valid date: %q", result.Content)
	}
}

func TestDatetimeTool_Execute_InvalidTimezone(t *testing.T) {
	tool := &DatetimeTool{}
	args, _ := json.Marshal(datetimeArgs{Timezone: "Invalid/Timezone"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for invalid timezone")
	}
	if !strings.Contains(result.Content, "invalid timezone") {
		t.Errorf("error message should mention invalid timezone, got: %q", result.Content)
	}
}

func TestDatetimeTool_Execute_NilArgs(t *testing.T) {
	tool := &DatetimeTool{}
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	// nil args should work, returning default format
	if result.IsError {
		t.Fatalf("Execute() returned error for nil args: %s", result.Content)
	}
}

func TestDatetimeTool_Execute_InvalidJSON(t *testing.T) {
	tool := &DatetimeTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for invalid JSON")
	}
}
