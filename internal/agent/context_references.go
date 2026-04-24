package agent

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var refPattern = regexp.MustCompile(`@(file|diff|staged|url):([^\s]+)`)

// ExpandContextReferences scans text for @file:, @diff:, @staged:, @url: references
// and replaces them with their actual content.
func ExpandContextReferences(text string) string {
	return refPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := refPattern.FindStringSubmatch(match)
		if len(parts) < 3 { return match }
		refType, refTarget := parts[1], parts[2]
		switch refType {
		case "file":
			data, err := os.ReadFile(refTarget)
			if err != nil { return fmt.Sprintf("[error reading %s: %v]", refTarget, err) }
			content := string(data)
			if len(content) > 50000 { content = content[:50000] + "\n... [truncated]" }
			return fmt.Sprintf("<file path=%q>\n%s\n</file>", refTarget, content)
		case "diff":
			out, err := exec.Command("git", "diff", refTarget).Output()
			if err != nil { return fmt.Sprintf("[error getting diff: %v]", err) }
			return fmt.Sprintf("<diff ref=%q>\n%s\n</diff>", refTarget, strings.TrimSpace(string(out)))
		case "staged":
			out, err := exec.Command("git", "diff", "--cached").Output()
			if err != nil { return fmt.Sprintf("[error getting staged: %v]", err) }
			return fmt.Sprintf("<staged>\n%s\n</staged>", strings.TrimSpace(string(out)))
		case "url":
			return fmt.Sprintf("[URL content: %s - fetch via browser tool]", refTarget)
		default:
			return match
		}
	})
}
