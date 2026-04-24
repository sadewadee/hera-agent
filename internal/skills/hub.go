package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/paths"
)

// Hub provides access to a remote skills registry for downloading and sharing skills.
type Hub struct {
	baseURL  string
	client   *http.Client
	cacheDir string
}

// HubSkill represents a skill available in the remote registry.
type HubSkill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Downloads   int      `json:"downloads"`
	Tags        []string `json:"tags"`
	URL         string   `json:"url"`
}

// NewHub creates a new Skills Hub client.
func NewHub(baseURL, cacheDir string) *Hub {
	if baseURL == "" {
		baseURL = "https://skills.hera.dev/api/v1"
	}
	if cacheDir == "" {
		cacheDir = filepath.Join(paths.UserSkills(), "hub-cache")
	}
	return &Hub{
		baseURL:  strings.TrimRight(baseURL, "/"),
		client:   &http.Client{Timeout: 30 * time.Second},
		cacheDir: cacheDir,
	}
}

// Search searches the hub for skills matching the query.
func (h *Hub) Search(ctx context.Context, query string) ([]HubSkill, error) {
	url := fmt.Sprintf("%s/skills/search?q=%s", h.baseURL, query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hub error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Skills []HubSkill `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Skills, nil
}

// Download fetches a skill from the hub by URL and saves it to the skills directory.
func (h *Hub) Download(ctx context.Context, skillURL, destDir string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, skillURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download error (status %d): %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return "", fmt.Errorf("read skill: %w", err)
	}

	// Extract filename from URL or Content-Disposition.
	filename := filepath.Base(skillURL)
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	// Ensure destination directory exists.
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	destPath := filepath.Join(destDir, filename)
	if err := os.WriteFile(destPath, content, 0o644); err != nil {
		return "", fmt.Errorf("write skill: %w", err)
	}

	return destPath, nil
}

// List retrieves all available skills from the hub.
func (h *Hub) List(ctx context.Context) ([]HubSkill, error) {
	url := fmt.Sprintf("%s/skills", h.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hub error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Skills []HubSkill `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Skills, nil
}

// Upload publishes a local skill to the hub.
func (h *Hub) Upload(ctx context.Context, skillPath, apiKey string) error {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return fmt.Errorf("read skill: %w", err)
	}

	url := fmt.Sprintf("%s/skills", h.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(content)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/markdown")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
