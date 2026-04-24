package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// RegexTool provides regex testing and extraction capabilities.
type RegexTool struct{}

type regexToolArgs struct {
	Action  string `json:"action"`
	Pattern string `json:"pattern"`
	Text    string `json:"text"`
	Replace string `json:"replace,omitempty"`
}

func (t *RegexTool) Name() string { return "regex" }

func (t *RegexTool) Description() string {
	return "Regex operations: test patterns, find matches, extract groups, and replace."
}

func (t *RegexTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["test", "find_all", "find_groups", "replace", "split"],
				"description": "Regex action: test (match?), find_all (all matches), find_groups (capture groups), replace, split."
			},
			"pattern": {
				"type": "string",
				"description": "Regular expression pattern (Go/RE2 syntax)."
			},
			"text": {
				"type": "string",
				"description": "Text to search or transform."
			},
			"replace": {
				"type": "string",
				"description": "Replacement string (for replace action). Supports $1, $2 group references."
			}
		},
		"required": ["action", "pattern", "text"]
	}`)
}

func (t *RegexTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a regexToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	re, err := regexp.Compile(a.Pattern)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid regex: %v", err), IsError: true}, nil
	}

	switch a.Action {
	case "test":
		if re.MatchString(a.Text) {
			loc := re.FindStringIndex(a.Text)
			return &tools.Result{Content: fmt.Sprintf("Match found at position %d-%d", loc[0], loc[1])}, nil
		}
		return &tools.Result{Content: "No match"}, nil

	case "find_all":
		matches := re.FindAllString(a.Text, -1)
		if len(matches) == 0 {
			return &tools.Result{Content: "No matches found"}, nil
		}
		out, _ := json.MarshalIndent(matches, "", "  ")
		return &tools.Result{Content: fmt.Sprintf("%d matches found:\n%s", len(matches), string(out))}, nil

	case "find_groups":
		matches := re.FindAllStringSubmatch(a.Text, -1)
		if len(matches) == 0 {
			return &tools.Result{Content: "No matches found"}, nil
		}
		var sb strings.Builder
		names := re.SubexpNames()
		for i, match := range matches {
			fmt.Fprintf(&sb, "Match %d:\n", i+1)
			for j, group := range match {
				name := ""
				if j < len(names) && names[j] != "" {
					name = fmt.Sprintf(" (%s)", names[j])
				}
				fmt.Fprintf(&sb, "  Group %d%s: %q\n", j, name, group)
			}
		}
		return &tools.Result{Content: sb.String()}, nil

	case "replace":
		result := re.ReplaceAllString(a.Text, a.Replace)
		return &tools.Result{Content: result}, nil

	case "split":
		parts := re.Split(a.Text, -1)
		out, _ := json.MarshalIndent(parts, "", "  ")
		return &tools.Result{Content: string(out)}, nil

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

// RegisterRegex registers the regex tool with the given registry.
func RegisterRegex(registry *tools.Registry) {
	registry.Register(&RegexTool{})
}
