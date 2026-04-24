package skills

import (
	"os"
	"path/filepath"
	"testing"
)

const validSkillContent = `---
name: test-skill
description: A test skill
triggers:
  - greet
  - hello
platforms:
  - cli
requires_tools:
  - web_search
---
# Test Skill

This is the body of the test skill.
`

const noFrontmatterContent = `# Just a plain markdown file
No frontmatter here.
`

func TestParseSkillFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-skill.md")
	if err := os.WriteFile(path, []byte(validSkillContent), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	skill, err := ParseSkillFile(path)
	if err != nil {
		t.Fatalf("ParseSkillFile() error = %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill" {
		t.Errorf("Description = %q, want %q", skill.Description, "A test skill")
	}
	if len(skill.Triggers) != 2 {
		t.Fatalf("Triggers length = %d, want 2", len(skill.Triggers))
	}
	if skill.Triggers[0] != "greet" || skill.Triggers[1] != "hello" {
		t.Errorf("Triggers = %v, want [greet hello]", skill.Triggers)
	}
	if len(skill.Platforms) != 1 || skill.Platforms[0] != "cli" {
		t.Errorf("Platforms = %v, want [cli]", skill.Platforms)
	}
	if len(skill.RequiresTools) != 1 || skill.RequiresTools[0] != "web_search" {
		t.Errorf("RequiresTools = %v, want [web_search]", skill.RequiresTools)
	}
	if skill.Content == "" {
		t.Error("Content is empty, expected body after frontmatter")
	}
	if skill.FilePath != path {
		t.Errorf("FilePath = %q, want %q", skill.FilePath, path)
	}
}

func TestParseSkillFile_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.md")
	if err := os.WriteFile(path, []byte(noFrontmatterContent), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := ParseSkillFile(path)
	if err == nil {
		t.Error("ParseSkillFile() expected error for file without frontmatter")
	}
}

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/some/dir", "/another/dir")
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
	if len(loader.dirs) != 2 {
		t.Errorf("dirs length = %d, want 2", len(loader.dirs))
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()

	// Create two valid skill files.
	skill1 := `---
name: alpha
description: Alpha skill
triggers:
  - a
---
Alpha body.
`
	skill2 := `---
name: beta
description: Beta skill
triggers:
  - b
---
Beta body.
`
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"), []byte(skill1), 0o644); err != nil {
		t.Fatalf("write alpha.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta.md"), []byte(skill2), 0o644); err != nil {
		t.Fatalf("write beta.md: %v", err)
	}
	// Also write a non-.md file to verify it is ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("write readme.txt: %v", err)
	}

	loader := NewLoader(dir)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	all := loader.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d skills, want 2", len(all))
	}

	names := make(map[string]bool)
	for _, s := range all {
		names[s.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("All() names = %v, want alpha and beta", names)
	}
}

func TestFindByTrigger(t *testing.T) {
	dir := t.TempDir()

	skill1 := `---
name: greeter
description: Greets
triggers:
  - greet
  - hello
---
Greet body.
`
	skill2 := `---
name: farewell
description: Says goodbye
triggers:
  - bye
  - goodbye
---
Bye body.
`
	os.WriteFile(filepath.Join(dir, "greeter.md"), []byte(skill1), 0o644)
	os.WriteFile(filepath.Join(dir, "farewell.md"), []byte(skill2), 0o644)

	loader := NewLoader(dir)
	loader.LoadAll()

	// Find by trigger "hello" should return greeter.
	found := loader.FindByTrigger("hello")
	if len(found) != 1 {
		t.Fatalf("FindByTrigger('hello') returned %d skills, want 1", len(found))
	}
	if found[0].Name != "greeter" {
		t.Errorf("FindByTrigger('hello') found %q, want %q", found[0].Name, "greeter")
	}

	// Case insensitive.
	found = loader.FindByTrigger("HELLO")
	if len(found) != 1 {
		t.Fatalf("FindByTrigger('HELLO') returned %d skills, want 1", len(found))
	}

	// No match.
	found = loader.FindByTrigger("nonexistent")
	if len(found) != 0 {
		t.Errorf("FindByTrigger('nonexistent') returned %d skills, want 0", len(found))
	}
}

func TestAll(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: only-one
description: The only skill
triggers:
  - one
---
Body.
`
	os.WriteFile(filepath.Join(dir, "only.md"), []byte(content), 0o644)

	loader := NewLoader(dir)
	loader.LoadAll()

	all := loader.All()
	if len(all) != 1 {
		t.Fatalf("All() returned %d skills, want 1", len(all))
	}
	if all[0].Name != "only-one" {
		t.Errorf("All()[0].Name = %q, want %q", all[0].Name, "only-one")
	}
}

func TestGet(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: finder
description: A findable skill
triggers:
  - find
---
Finder body.
`
	os.WriteFile(filepath.Join(dir, "finder.md"), []byte(content), 0o644)

	loader := NewLoader(dir)
	loader.LoadAll()

	skill, ok := loader.Get("finder")
	if !ok {
		t.Fatal("Get('finder') returned false")
	}
	if skill.Name != "finder" {
		t.Errorf("Name = %q, want %q", skill.Name, "finder")
	}

	_, ok = loader.Get("nonexistent")
	if ok {
		t.Error("Get('nonexistent') returned true")
	}
}
