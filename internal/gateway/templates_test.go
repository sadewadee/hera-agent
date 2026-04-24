package gateway

import (
	"strings"
	"testing"
)

func TestTemplateEngine_Render(t *testing.T) {
	te := NewTemplateEngine()

	result, err := te.Render("welcome", map[string]string{
		"username":   "Alice",
		"agent_name": "Hera",
	})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("result should contain 'Alice', got: %s", result)
	}
	if !strings.Contains(result, "Hera") {
		t.Errorf("result should contain 'Hera', got: %s", result)
	}
}

func TestTemplateEngine_CustomTemplate(t *testing.T) {
	te := NewTemplateEngine()

	te.Register("custom", "Hello {{name}}, you have {{count}} messages.")
	result, err := te.Render("custom", map[string]string{
		"name":  "Bob",
		"count": "5",
	})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if result != "Hello Bob, you have 5 messages." {
		t.Errorf("got %q", result)
	}
}

func TestTemplateEngine_MissingTemplate(t *testing.T) {
	te := NewTemplateEngine()

	_, err := te.Render("nonexistent", nil)
	if err == nil {
		t.Error("expected error for missing template")
	}
}

func TestTemplateEngine_List(t *testing.T) {
	te := NewTemplateEngine()

	names := te.List()
	if len(names) == 0 {
		t.Error("default templates should be registered")
	}

	// Check that default templates exist
	defaults := map[string]bool{"welcome": false, "error": false, "rate_limited": false}
	for _, name := range names {
		if _, ok := defaults[name]; ok {
			defaults[name] = true
		}
	}
	for name, found := range defaults {
		if !found {
			t.Errorf("missing default template: %s", name)
		}
	}
}

func TestTemplateEngine_Delete(t *testing.T) {
	te := NewTemplateEngine()

	te.Register("temp", "temporary")
	te.Delete("temp")

	_, ok := te.Get("temp")
	if ok {
		t.Error("template should be deleted")
	}
}

func TestTemplateEngine_UnreplacedVars(t *testing.T) {
	te := NewTemplateEngine()

	te.Register("partial", "Hello {{name}}, your {{status}} is ready.")
	result, _ := te.Render("partial", map[string]string{"name": "Alice"})

	if !strings.Contains(result, "{{status}}") {
		t.Error("unreplaced vars should remain in output")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("replaced vars should be substituted")
	}
}
