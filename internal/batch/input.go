package batch

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// PromptSource provides prompts for a batch run.
type PromptSource interface {
	// Prompts returns the full list of prompts. May be called multiple times.
	Prompts() ([]string, error)
}

// FileSource reads one prompt per non-empty line from a file.
type FileSource struct {
	Path string
}

// NewFileSource creates a FileSource for the given file path.
func NewFileSource(path string) *FileSource {
	return &FileSource{Path: path}
}

// Prompts reads and returns all non-empty, non-comment lines from the file.
func (s *FileSource) Prompts() ([]string, error) {
	f, err := os.Open(s.Path)
	if err != nil {
		return nil, fmt.Errorf("open prompt file %q: %w", s.Path, err)
	}
	defer f.Close()
	return readPrompts(f)
}

// StdinSource reads one prompt per non-empty line from stdin.
type StdinSource struct {
	r io.Reader // injectable for testing; defaults to os.Stdin
}

// NewStdinSource creates a StdinSource reading from os.Stdin.
func NewStdinSource() *StdinSource {
	return &StdinSource{r: os.Stdin}
}

// Prompts reads and returns all non-empty lines from stdin.
func (s *StdinSource) Prompts() ([]string, error) {
	r := s.r
	if r == nil {
		r = os.Stdin
	}
	return readPrompts(r)
}

// SliceSource serves prompts from an in-memory slice. Used in tests.
type SliceSource struct {
	items []string
}

// NewSliceSource creates a PromptSource backed by the provided prompts.
func NewSliceSource(prompts []string) *SliceSource {
	return &SliceSource{items: prompts}
}

// Prompts returns the slice's contents.
func (s *SliceSource) Prompts() ([]string, error) {
	return s.items, nil
}

// readPrompts reads lines from r, stripping blank lines and lines beginning
// with '#' (shell-style comments).
func readPrompts(r io.Reader) ([]string, error) {
	var prompts []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prompts = append(prompts, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan prompts: %w", err)
	}
	return prompts, nil
}
