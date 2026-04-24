package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSkinEngine_HasBundledSkins(t *testing.T) {
	engine := NewSkinEngine()

	expectedSkins := []string{"default", "midnight", "solar", "matrix", "rose"}
	for _, name := range expectedSkins {
		found := false
		for _, s := range engine.List() {
			if s == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected bundled skin %q not found", name)
		}
	}
}

func TestSkinEngine_Current_DefaultSkin(t *testing.T) {
	engine := NewSkinEngine()
	current := engine.Current()
	if current == nil {
		t.Fatal("Current() returned nil")
	}
	if current.Name != "default" {
		t.Errorf("Current().Name = %q, want %q", current.Name, "default")
	}
}

func TestSkinEngine_Set(t *testing.T) {
	engine := NewSkinEngine()

	if err := engine.Set("midnight"); err != nil {
		t.Fatalf("Set('midnight') error = %v", err)
	}
	current := engine.Current()
	if current.Name != "midnight" {
		t.Errorf("Current().Name = %q, want %q after Set", current.Name, "midnight")
	}
}

func TestSkinEngine_Set_NotFound(t *testing.T) {
	engine := NewSkinEngine()
	if err := engine.Set("nonexistent"); err == nil {
		t.Error("Set('nonexistent') should return error")
	}
}

func TestSkinEngine_LoadFromFile(t *testing.T) {
	dir := t.TempDir()
	skinPath := filepath.Join(dir, "custom.yaml")
	skinContent := `name: custom
description: Custom test skin
colors:
  prompt: red
  response: blue
  error: yellow
  info: green
  code: cyan
  muted: gray
  accent: magenta
  banner: white
`
	if err := os.WriteFile(skinPath, []byte(skinContent), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := NewSkinEngine()
	if err := engine.LoadFromFile(skinPath); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if err := engine.Set("custom"); err != nil {
		t.Fatalf("Set('custom') after LoadFromFile error = %v", err)
	}

	current := engine.Current()
	if current.Name != "custom" {
		t.Errorf("Current().Name = %q, want %q", current.Name, "custom")
	}
	if current.Colors.Prompt != "red" {
		t.Errorf("Colors.Prompt = %q, want %q", current.Colors.Prompt, "red")
	}
}

func TestSkinEngine_LoadFromDir(t *testing.T) {
	dir := t.TempDir()

	// Create two skin files
	for _, name := range []string{"alpha", "beta"} {
		skinContent := `name: ` + name + `
description: Test skin ` + name + `
colors:
  prompt: red
  response: white
  error: red
  info: blue
  code: green
  muted: gray
  accent: magenta
  banner: cyan
`
		if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(skinContent), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	engine := NewSkinEngine()
	if err := engine.LoadFromDir(dir); err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}

	// Should now have alpha and beta in addition to bundled skins
	if err := engine.Set("alpha"); err != nil {
		t.Errorf("Set('alpha') after LoadFromDir error = %v", err)
	}
	if err := engine.Set("beta"); err != nil {
		t.Errorf("Set('beta') after LoadFromDir error = %v", err)
	}
}

func TestSkinEngine_LoadFromDir_NonexistentDir(t *testing.T) {
	engine := NewSkinEngine()
	// Should not return error for nonexistent directory
	if err := engine.LoadFromDir("/nonexistent/dir"); err != nil {
		t.Errorf("LoadFromDir() for nonexistent dir should not error, got: %v", err)
	}
}

func TestSkin_ColorsPopulated(t *testing.T) {
	engine := NewSkinEngine()

	for _, name := range engine.List() {
		t.Run(name, func(t *testing.T) {
			if err := engine.Set(name); err != nil {
				t.Fatalf("Set(%q) error = %v", name, err)
			}
			skin := engine.Current()
			if skin.Colors.Prompt == "" {
				t.Errorf("skin %q: Colors.Prompt is empty", name)
			}
			if skin.Colors.Response == "" {
				t.Errorf("skin %q: Colors.Response is empty", name)
			}
			if skin.Colors.Error == "" {
				t.Errorf("skin %q: Colors.Error is empty", name)
			}
		})
	}
}
