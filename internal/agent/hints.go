package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// hintFileNames lists the filenames to look for when loading subdirectory hints.
// These are checked in the working directory and each parent up to the root.
var hintFileNames = []string{
	".hera.md",
	"AGENTS.md",
	"CLAUDE.md",
}

// LoadSubdirectoryHints scans the given directory and its parents for hint files
// (.hera.md, AGENTS.md, CLAUDE.md) and returns their combined content as context
// to inject into the system prompt.
func LoadSubdirectoryHints(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	// Resolve to absolute path.
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	var parts []string
	seen := make(map[string]bool)

	// Walk from the given directory up to the filesystem root.
	current := absDir
	for {
		for _, name := range hintFileNames {
			path := filepath.Join(current, name)
			if seen[path] {
				continue
			}
			seen[path] = true

			data, err := os.ReadFile(path)
			if err != nil {
				continue // file doesn't exist or unreadable
			}

			content := strings.TrimSpace(string(data))
			if content == "" {
				continue
			}

			// Tag with the source path for context.
			parts = append(parts, "<!-- from "+path+" -->\n"+content)
		}

		parent := filepath.Dir(current)
		if parent == current {
			break // reached root
		}
		current = parent
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, "\n\n"), nil
}
