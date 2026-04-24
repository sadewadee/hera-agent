package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHub_Search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/skills/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("missing query parameter 'q'")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"skills": []HubSkill{
				{Name: "web-search", Description: "Search the web", Author: "alice", Version: "1.0.0", Downloads: 42, Tags: []string{"search"}, URL: "https://example.com/web-search.md"},
				{Name: "code-review", Description: "Review code", Author: "bob", Version: "2.0.0", Downloads: 100, Tags: []string{"code"}, URL: "https://example.com/code-review.md"},
			},
		})
	}))
	defer srv.Close()

	hub := NewHub(srv.URL, t.TempDir())
	results, err := hub.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}
	if results[0].Name != "web-search" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "web-search")
	}
	if results[1].Author != "bob" {
		t.Errorf("results[1].Author = %q, want %q", results[1].Author, "bob")
	}
}

func TestHub_Download(t *testing.T) {
	skillContent := `---
name: downloaded-skill
description: A skill from the hub
triggers:
  - dl
---
Downloaded skill body.
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(skillContent))
	}))
	defer srv.Close()

	destDir := t.TempDir()
	hub := NewHub(srv.URL, t.TempDir())

	path, err := hub.Download(context.Background(), srv.URL+"/skills/downloaded-skill.md", destDir)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	// Verify the file was written.
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("downloaded file does not exist at %s: %v", path, statErr)
	}

	// Verify file content.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != skillContent {
		t.Errorf("downloaded content mismatch:\ngot:  %q\nwant: %q", string(data), skillContent)
	}

	// Verify filename ends in .md.
	if filepath.Ext(path) != ".md" {
		t.Errorf("downloaded path %q does not end in .md", path)
	}
}

func TestHub_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/skills" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"skills": []HubSkill{
				{Name: "skill-a", Description: "Skill A", Author: "charlie", Version: "1.0.0"},
				{Name: "skill-b", Description: "Skill B", Author: "dave", Version: "1.1.0"},
				{Name: "skill-c", Description: "Skill C", Author: "eve", Version: "0.9.0"},
			},
		})
	}))
	defer srv.Close()

	hub := NewHub(srv.URL, t.TempDir())
	results, err := hub.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("List() returned %d results, want 3", len(results))
	}

	names := make(map[string]bool)
	for _, s := range results {
		names[s.Name] = true
	}
	for _, want := range []string{"skill-a", "skill-b", "skill-c"} {
		if !names[want] {
			t.Errorf("List() missing %q", want)
		}
	}
}
