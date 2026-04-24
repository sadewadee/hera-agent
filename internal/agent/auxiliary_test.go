package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"no tags", "hello world", "hello world"},
		{"simple tags", "<p>hello</p>", " hello "},
		{"nested tags", "<div><b>bold</b></div>", "  bold  "},
		{"empty string", "", ""},
		{"self closing", "text<br/>more", "text more"},
		{"attributes", `<a href="url">link</a>`, " link "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTMLTags(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"no excess whitespace", "hello world", "hello world"},
		{"multiple spaces", "hello   world", "hello world"},
		{"tabs and newlines", "hello\t\n\tworld", "hello world"},
		{"leading trailing", "  hello  ", "hello"},
		{"empty string", "", ""},
		{"only whitespace", "   \t\n  ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collapseWhitespace(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		max    int
		expect string
	}{
		{"short text", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncation", "hello world long", 10, "hello w..."},
		{"empty", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.input, tt.max)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestAuxiliaryClient_NewAuxiliaryClient(t *testing.T) {
	provider := &fakeTitleProvider{response: "test"}
	client := NewAuxiliaryClient(provider)
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
}
