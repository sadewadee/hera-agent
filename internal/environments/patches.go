package environments

import (
	"fmt"
	"os"
	"strings"
)

// PatchOperation represents a file patch operation.
type PatchOperation struct {
	FilePath string
	Original []byte
	Modified []byte
}

// ApplyPatch writes the modified content to the file.
func (p *PatchOperation) Apply() error {
	return os.WriteFile(p.FilePath, p.Modified, 0644)
}

// Revert restores the original content.
func (p *PatchOperation) Revert() error {
	return os.WriteFile(p.FilePath, p.Original, 0644)
}

// ParseUnifiedDiff parses a unified diff string into patch operations (simplified).
func ParseUnifiedDiff(diff string) []PatchOperation {
	var patches []PatchOperation
	lines := strings.Split(diff, "\n")
	var currentFile string
	for _, line := range lines {
		if strings.HasPrefix(line, "--- a/") {
			currentFile = strings.TrimPrefix(line, "--- a/")
		} else if strings.HasPrefix(line, "+++ b/") && currentFile != "" {
			patches = append(patches, PatchOperation{FilePath: currentFile})
			currentFile = ""
		}
	}
	_ = fmt.Sprintf // keep import
	return patches
}
