package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// DatetimeTool returns the current date and time in various formats.
type DatetimeTool struct{}

type datetimeArgs struct {
	Format   string `json:"format,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

func (d *DatetimeTool) Name() string {
	return "datetime"
}

func (d *DatetimeTool) Description() string {
	return "Returns the current date and time. Supports custom format strings and timezones."
}

func (d *DatetimeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"format": {
				"type": "string",
				"description": "Output format: 'iso8601', 'unix', 'rfc3339', 'date', 'time', or a Go time format string. Defaults to 'iso8601'."
			},
			"timezone": {
				"type": "string",
				"description": "IANA timezone name (e.g. 'America/New_York', 'Europe/London'). Defaults to local timezone."
			}
		}
	}`)
}

func (d *DatetimeTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params datetimeArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
		}
	}

	now := time.Now()

	if params.Timezone != "" {
		loc, err := time.LoadLocation(params.Timezone)
		if err != nil {
			return &tools.Result{
				Content: fmt.Sprintf("invalid timezone %q: %v", params.Timezone, err),
				IsError: true,
			}, nil
		}
		now = now.In(loc)
	}

	format := params.Format
	if format == "" {
		format = "iso8601"
	}

	var output string
	switch format {
	case "iso8601", "rfc3339":
		output = now.Format(time.RFC3339)
	case "unix":
		output = fmt.Sprintf("%d", now.Unix())
	case "date":
		output = now.Format("2006-01-02")
	case "time":
		output = now.Format("15:04:05")
	default:
		output = now.Format(format)
	}

	return &tools.Result{Content: output}, nil
}

// RegisterDatetime registers the datetime tool with the given registry.
func RegisterDatetime(registry *tools.Registry) {
	registry.Register(&DatetimeTool{})
}
