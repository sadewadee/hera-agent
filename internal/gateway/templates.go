package gateway

import (
	"fmt"
	"strings"
	"sync"
)

// Template represents a response template with variable substitution.
type Template struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Format  string `json:"format,omitempty"` // "markdown", "plain", "html"
}

// TemplateEngine manages response templates for the gateway.
type TemplateEngine struct {
	mu        sync.RWMutex
	templates map[string]*Template
}

// NewTemplateEngine creates a new template engine.
func NewTemplateEngine() *TemplateEngine {
	te := &TemplateEngine{
		templates: make(map[string]*Template),
	}
	te.registerDefaults()
	return te
}

// Register adds a template to the engine.
func (te *TemplateEngine) Register(name, content string) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.templates[name] = &Template{Name: name, Content: content}
}

// Get retrieves a template by name.
func (te *TemplateEngine) Get(name string) (*Template, bool) {
	te.mu.RLock()
	defer te.mu.RUnlock()
	t, ok := te.templates[name]
	return t, ok
}

// Render renders a template with the given variables.
// Variables are substituted using {{key}} syntax.
func (te *TemplateEngine) Render(name string, vars map[string]string) (string, error) {
	te.mu.RLock()
	tmpl, ok := te.templates[name]
	te.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}

	result := tmpl.Content
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// List returns all registered template names.
func (te *TemplateEngine) List() []string {
	te.mu.RLock()
	defer te.mu.RUnlock()
	names := make([]string, 0, len(te.templates))
	for name := range te.templates {
		names = append(names, name)
	}
	return names
}

// Delete removes a template by name.
func (te *TemplateEngine) Delete(name string) {
	te.mu.Lock()
	defer te.mu.Unlock()
	delete(te.templates, name)
}

func (te *TemplateEngine) registerDefaults() {
	te.templates["welcome"] = &Template{
		Name:    "welcome",
		Content: "Welcome, {{username}}! I'm {{agent_name}}, your AI assistant. How can I help you today?",
	}
	te.templates["error"] = &Template{
		Name:    "error",
		Content: "I encountered an error processing your request: {{error_message}}. Please try again.",
	}
	te.templates["rate_limited"] = &Template{
		Name:    "rate_limited",
		Content: "You've sent too many messages. Please wait {{wait_time}} before trying again.",
	}
	te.templates["maintenance"] = &Template{
		Name:    "maintenance",
		Content: "I'm currently undergoing maintenance and will be back shortly. Expected return: {{eta}}.",
	}
	te.templates["goodbye"] = &Template{
		Name:    "goodbye",
		Content: "Goodbye, {{username}}! Feel free to come back anytime.",
	}
}
