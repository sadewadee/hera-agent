package swe

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DiffOp is the operation type for a diff line.
type DiffOp int

const (
	// OpContext is an unchanged context line (no prefix or space prefix).
	OpContext DiffOp = iota
	// OpAdd is a line added in the new file (+ prefix).
	OpAdd
	// OpDelete is a line removed from the old file (- prefix).
	OpDelete
)

// DiffLine represents a single line in a hunk.
type DiffLine struct {
	Op   DiffOp
	Text string // line content without the +/-/space prefix
}

// DiffHunk represents a single @@ block in a unified diff.
type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// ParseHunks parses a unified diff string into hunks.
// It tolerates the --- and +++ header lines and skips them.
// Exported for testing.
func ParseHunks(diff string) ([]DiffHunk, error) {
	var hunks []DiffHunk
	var current *DiffHunk

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "@@ ") {
			// Flush previous hunk.
			if current != nil {
				hunks = append(hunks, *current)
			}
			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, fmt.Errorf("parse hunk header %q: %w", line, err)
			}
			current = &hunk
			continue
		}

		// Skip file header lines.
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") {
			continue
		}

		if current == nil {
			// Lines before the first hunk header are ignored.
			continue
		}

		if len(line) == 0 {
			// Empty line treated as context.
			current.Lines = append(current.Lines, DiffLine{Op: OpContext, Text: ""})
			continue
		}

		switch line[0] {
		case '+':
			current.Lines = append(current.Lines, DiffLine{Op: OpAdd, Text: line[1:]})
		case '-':
			current.Lines = append(current.Lines, DiffLine{Op: OpDelete, Text: line[1:]})
		case ' ':
			current.Lines = append(current.Lines, DiffLine{Op: OpContext, Text: line[1:]})
		case '\\':
			// "\ No newline at end of file" — skip.
		default:
			current.Lines = append(current.Lines, DiffLine{Op: OpContext, Text: line})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan diff: %w", err)
	}
	if current != nil {
		hunks = append(hunks, *current)
	}
	return hunks, nil
}

// parseHunkHeader parses a line of the form "@@ -a,b +c,d @@ optional".
// The count fields default to 1 if omitted.
func parseHunkHeader(line string) (DiffHunk, error) {
	// Extract the @@ … @@ part.
	end := strings.Index(line[3:], "@@")
	var rangeStr string
	if end >= 0 {
		rangeStr = strings.TrimSpace(line[3 : 3+end])
	} else {
		rangeStr = strings.TrimSpace(line[3:])
	}

	parts := strings.Fields(rangeStr) // [-a,b +c,d]
	if len(parts) < 2 {
		return DiffHunk{}, fmt.Errorf("expected 2 range fields, got %d", len(parts))
	}

	oldStart, oldCount, err := parseRange(parts[0])
	if err != nil {
		return DiffHunk{}, fmt.Errorf("old range: %w", err)
	}
	newStart, newCount, err := parseRange(parts[1])
	if err != nil {
		return DiffHunk{}, fmt.Errorf("new range: %w", err)
	}

	return DiffHunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
	}, nil
}

// parseRange parses "-a,b" or "+a,b" into (start, count).
// Count defaults to 1 if the comma form is absent.
func parseRange(s string) (start, count int, err error) {
	if len(s) == 0 {
		return 0, 0, fmt.Errorf("empty range")
	}
	// Strip leading - or +
	s = s[1:]
	if comma := strings.Index(s, ","); comma >= 0 {
		start, err = strconv.Atoi(s[:comma])
		if err != nil {
			return 0, 0, fmt.Errorf("parse start %q: %w", s[:comma], err)
		}
		count, err = strconv.Atoi(s[comma+1:])
		if err != nil {
			return 0, 0, fmt.Errorf("parse count %q: %w", s[comma+1:], err)
		}
		return start, count, nil
	}
	start, err = strconv.Atoi(s)
	if err != nil {
		return 0, 0, fmt.Errorf("parse start %q: %w", s, err)
	}
	return start, 1, nil
}

// ApplyUnifiedDiff applies a unified diff to the file at filePath.
// This is a real line-based implementation; the stub in
// internal/tools/builtin/patch.go only counts lines and does not write.
// Returns an error if any hunk cannot be applied cleanly.
func ApplyUnifiedDiff(filePath, diff string) error {
	hunks, err := ParseHunks(diff)
	if err != nil {
		return fmt.Errorf("parse diff: %w", err)
	}
	if len(hunks) == 0 {
		return nil // nothing to apply
	}

	// Read existing file contents (if the file does not exist, treat as empty).
	var fileLines []string
	data, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read file %s: %w", filePath, err)
	}
	if err == nil {
		raw := strings.Split(string(data), "\n")
		// If the file ends with a newline, Split produces an empty trailing element.
		// We keep it because hunk offsets are 1-based and must match.
		fileLines = raw
	}

	result, err := applyHunks(fileLines, hunks)
	if err != nil {
		return err
	}

	output := strings.Join(result, "\n")
	if err := os.WriteFile(filePath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", filePath, err)
	}
	return nil
}

// applyHunks applies all hunks sequentially to lines and returns the result.
func applyHunks(lines []string, hunks []DiffHunk) ([]string, error) {
	result := make([]string, 0, len(lines)+16)
	srcIdx := 0 // 0-based index into lines (1-based in diff offsets)

	for _, hunk := range hunks {
		// OldStart is 1-based; a value of 0 means the file was empty.
		hunkStart := hunk.OldStart - 1
		if hunk.OldStart == 0 {
			hunkStart = 0
		}

		// Copy lines before this hunk.
		if hunkStart > len(lines) {
			return nil, fmt.Errorf("hunk @@ -%d,%d +%d,%d @@ starts past end of file (%d lines)",
				hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount, len(lines))
		}
		for srcIdx < hunkStart {
			result = append(result, lines[srcIdx])
			srcIdx++
		}

		// Apply the hunk.
		for _, dl := range hunk.Lines {
			switch dl.Op {
			case OpContext:
				if srcIdx >= len(lines) {
					return nil, fmt.Errorf("context line %q at position %d exceeds file length %d",
						dl.Text, srcIdx+1, len(lines))
				}
				if lines[srcIdx] != dl.Text {
					return nil, fmt.Errorf("context mismatch at line %d: file has %q, diff expects %q",
						srcIdx+1, lines[srcIdx], dl.Text)
				}
				result = append(result, lines[srcIdx])
				srcIdx++
			case OpDelete:
				if srcIdx >= len(lines) {
					return nil, fmt.Errorf("delete line %q at position %d exceeds file length %d",
						dl.Text, srcIdx+1, len(lines))
				}
				if lines[srcIdx] != dl.Text {
					return nil, fmt.Errorf("delete mismatch at line %d: file has %q, diff expects %q",
						srcIdx+1, lines[srcIdx], dl.Text)
				}
				srcIdx++ // consume the old line without writing it
			case OpAdd:
				result = append(result, dl.Text)
			}
		}
	}

	// Copy any remaining lines after all hunks.
	for srcIdx < len(lines) {
		result = append(result, lines[srcIdx])
		srcIdx++
	}

	return result, nil
}
