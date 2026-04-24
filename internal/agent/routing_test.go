package agent

import (
	"testing"

	"github.com/sadewadee/hera/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestClassifyQuery_Simple(t *testing.T) {
	cfg := DefaultRoutingConfig()
	tests := []struct {
		name  string
		input string
		want  QueryComplexity
	}{
		{"greeting", "hello", ComplexitySimple},
		{"short_question", "what time is it?", ComplexitySimple},
		{"thanks", "thank you", ComplexitySimple},
		{"yes", "yes", ComplexitySimple},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ClassifyQuery(tt.input, cfg))
		})
	}
}

func TestClassifyQuery_Complex_LongText(t *testing.T) {
	cfg := DefaultRoutingConfig()
	// More than 28 words.
	longText := "Please help me understand how to implement a complex distributed system " +
		"that handles high throughput with low latency while maintaining strong consistency " +
		"guarantees across multiple data centers"
	assert.Equal(t, ComplexityComplex, ClassifyQuery(longText, cfg))
}

func TestClassifyQuery_Complex_CodeMarkers(t *testing.T) {
	cfg := DefaultRoutingConfig()
	tests := []struct {
		name  string
		input string
	}{
		{"backticks", "Fix the `main` function"},
		{"code_block", "```go\nfunc main() {}\n```"},
		{"func", "write a func that"},
		{"function", "create a function for"},
		{"def", "write a def parse"},
		{"class", "implement class User"},
		{"import", "fix this import statement"},
		{"package", "add package main"},
		{"console_log", "why does console.log fail"},
		{"if_err", "handle if err != nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ComplexityComplex, ClassifyQuery(tt.input, cfg))
		})
	}
}

func TestClassifyQuery_Complex_ComplexPatterns(t *testing.T) {
	cfg := DefaultRoutingConfig()
	tests := []struct {
		name  string
		input string
	}{
		{"explain", "explain how goroutines work"},
		{"analyze", "analyze this error log"},
		{"compare", "compare Go vs Rust"},
		{"implement", "implement a queue"},
		{"refactor", "refactor this code"},
		{"debug", "debug the crash"},
		{"optimize", "optimize this query"},
		{"architecture", "review the architecture"},
		{"design_pattern", "use a design pattern"},
		{"algorithm", "implement sorting algorithm"},
		{"write_a", "write a server"},
		{"create_a", "create a database schema"},
		{"build_a", "build a REST API"},
		{"step_by_step", "explain step by step"},
		{"comprehensive", "give comprehensive overview"},
		{"pros_and_cons", "list pros and cons"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ComplexityComplex, ClassifyQuery(tt.input, cfg))
		})
	}
}

func TestClassifyQuery_CustomThreshold(t *testing.T) {
	cfg := RoutingConfig{
		Enabled:             true,
		SimpleWordThreshold: 5,
	}
	assert.Equal(t, ComplexitySimple, ClassifyQuery("hello there", cfg))
	assert.Equal(t, ComplexityComplex, ClassifyQuery("this is a slightly longer sentence here", cfg))
}

func TestClassifyQuery_ZeroThreshold(t *testing.T) {
	cfg := RoutingConfig{
		Enabled:             true,
		SimpleWordThreshold: 0, // should default to 28
	}
	assert.Equal(t, ComplexitySimple, ClassifyQuery("hello", cfg))
}

func TestRouteModel_Enabled(t *testing.T) {
	cfg := RoutingConfig{
		Enabled:      true,
		CheapModel:   "gpt-4o-mini",
		CapableModel: "gpt-4o",
	}

	simple := RouteModel("hello", cfg)
	assert.Equal(t, "gpt-4o-mini", simple)

	complex := RouteModel("explain how distributed systems achieve consensus", cfg)
	assert.Equal(t, "gpt-4o", complex)
}

func TestRouteModel_Disabled(t *testing.T) {
	cfg := RoutingConfig{
		Enabled:      false,
		CheapModel:   "gpt-4o-mini",
		CapableModel: "gpt-4o",
	}
	assert.Equal(t, "", RouteModel("hello", cfg))
}

func TestRouteModel_EmptyModels(t *testing.T) {
	cfg := RoutingConfig{
		Enabled:      true,
		CheapModel:   "",
		CapableModel: "",
	}
	assert.Equal(t, "", RouteModel("hello", cfg))
	assert.Equal(t, "", RouteModel("explain distributed systems in detail", cfg))
}

func TestDefaultRoutingConfig(t *testing.T) {
	cfg := DefaultRoutingConfig()
	assert.True(t, cfg.Enabled)
	assert.Equal(t, 28, cfg.SimpleWordThreshold)
	assert.Equal(t, "gpt-4o-mini", cfg.CheapModel)
	assert.Equal(t, "gpt-4o", cfg.CapableModel)
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single", "hello", 1},
		{"two", "hello world", 2},
		{"with_extra_spaces", "  hello   world  ", 2},
		{"tabs", "hello\tworld", 2},
		{"newlines", "hello\nworld", 2},
		{"mixed_whitespace", "  a  b\tc\nd  ", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countWords(tt.input))
		})
	}
}

func TestContainsCodeMarkers(t *testing.T) {
	assert.True(t, containsCodeMarkers("use func main"))
	assert.True(t, containsCodeMarkers("fix this `bug`"))
	assert.True(t, containsCodeMarkers("import os"))
	assert.False(t, containsCodeMarkers("hello world"))
}

func TestContainsComplexPatterns(t *testing.T) {
	assert.True(t, containsComplexPatterns("explain this"))
	assert.True(t, containsComplexPatterns("analyze the data"))
	assert.True(t, containsComplexPatterns("give me a comprehensive review"))
	assert.False(t, containsComplexPatterns("hello world"))
}

// TestRoutingConfigFromConfig verifies that user config values override defaults
// and that missing fields fall back to DefaultRoutingConfig values.
func TestRoutingConfigFromConfig(t *testing.T) {
	defaults := DefaultRoutingConfig()

	tests := []struct {
		name    string
		input   config.RoutingConfig
		wantCfg RoutingConfig
	}{
		{
			name:  "empty config uses defaults",
			input: config.RoutingConfig{},
			wantCfg: RoutingConfig{
				Enabled:             defaults.Enabled,
				CheapModel:          defaults.CheapModel,
				CapableModel:        defaults.CapableModel,
				SimpleWordThreshold: defaults.SimpleWordThreshold,
			},
		},
		{
			name: "user cheap model overrides default",
			input: config.RoutingConfig{
				CheapModel: "custom-small",
			},
			wantCfg: RoutingConfig{
				Enabled:             defaults.Enabled,
				CheapModel:          "custom-small",
				CapableModel:        defaults.CapableModel,
				SimpleWordThreshold: defaults.SimpleWordThreshold,
			},
		},
		{
			name: "user capable model overrides default",
			input: config.RoutingConfig{
				CapableModel: "custom-large",
			},
			wantCfg: RoutingConfig{
				Enabled:             defaults.Enabled,
				CheapModel:          defaults.CheapModel,
				CapableModel:        "custom-large",
				SimpleWordThreshold: defaults.SimpleWordThreshold,
			},
		},
		{
			name: "both models overridden",
			input: config.RoutingConfig{
				CheapModel:   "my-cheap",
				CapableModel: "my-capable",
			},
			wantCfg: RoutingConfig{
				Enabled:             defaults.Enabled,
				CheapModel:          "my-cheap",
				CapableModel:        "my-capable",
				SimpleWordThreshold: defaults.SimpleWordThreshold,
			},
		},
		{
			name: "short_threshold_chars overrides word threshold",
			input: config.RoutingConfig{
				ShortThresholdChars: 50,
			},
			wantCfg: RoutingConfig{
				Enabled:             defaults.Enabled,
				CheapModel:          defaults.CheapModel,
				CapableModel:        defaults.CapableModel,
				SimpleWordThreshold: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoutingConfigFromConfig(&tt.input)
			assert.Equal(t, tt.wantCfg.Enabled, got.Enabled)
			assert.Equal(t, tt.wantCfg.CheapModel, got.CheapModel)
			assert.Equal(t, tt.wantCfg.CapableModel, got.CapableModel)
			assert.Equal(t, tt.wantCfg.SimpleWordThreshold, got.SimpleWordThreshold)
		})
	}
}
