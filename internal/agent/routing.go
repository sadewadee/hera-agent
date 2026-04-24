package agent

import (
	"strings"
	"unicode"

	"github.com/sadewadee/hera/internal/config"
)

// QueryComplexity represents the classified complexity of a user query.
type QueryComplexity int

const (
	// ComplexitySimple indicates a short, simple query suitable for a cheaper model.
	ComplexitySimple QueryComplexity = iota
	// ComplexityComplex indicates a complex query that needs a capable model.
	ComplexityComplex
)

// RoutingConfig configures the smart model routing behavior.
type RoutingConfig struct {
	// Enabled activates smart model routing.
	Enabled bool `json:"enabled" yaml:"enabled"`
	// SimpleWordThreshold is the max word count for a "simple" query (default 28).
	SimpleWordThreshold int `json:"simple_word_threshold" yaml:"simple_word_threshold"`
	// CheapModel is the model name used for simple queries.
	CheapModel string `json:"cheap_model" yaml:"cheap_model"`
	// CapableModel is the model name used for complex queries.
	CapableModel string `json:"capable_model" yaml:"capable_model"`
}

// DefaultRoutingConfig returns the default routing configuration.
func DefaultRoutingConfig() RoutingConfig {
	return RoutingConfig{
		Enabled:             true,
		SimpleWordThreshold: 28,
		CheapModel:          "gpt-4o-mini",
		CapableModel:        "gpt-4o",
	}
}

// RoutingConfigFromConfig merges user-supplied routing config over the defaults.
// Only non-zero values in cfg override the defaults, so any unset field falls
// back to DefaultRoutingConfig. This lets operators override just one model
// without having to redeclare the full config block.
func RoutingConfigFromConfig(cfg *config.RoutingConfig) RoutingConfig {
	rc := DefaultRoutingConfig()
	if cfg == nil {
		return rc
	}
	if cfg.CheapModel != "" {
		rc.CheapModel = cfg.CheapModel
	}
	if cfg.CapableModel != "" {
		rc.CapableModel = cfg.CapableModel
	}
	if cfg.ShortThresholdChars > 0 {
		rc.SimpleWordThreshold = cfg.ShortThresholdChars
	}
	return rc
}

// ClassifyQuery determines the complexity of a user query.
// A query is "simple" if:
//   - word count <= SimpleWordThreshold (default 28)
//   - does not contain code markers (backticks, code keywords)
//   - does not contain complex request indicators
func ClassifyQuery(text string, cfg RoutingConfig) QueryComplexity {
	threshold := cfg.SimpleWordThreshold
	if threshold <= 0 {
		threshold = 28
	}

	words := countWords(text)

	// Long queries are complex.
	if words > threshold {
		return ComplexityComplex
	}

	// Check for code indicators.
	if containsCodeMarkers(text) {
		return ComplexityComplex
	}

	// Check for complex request patterns.
	if containsComplexPatterns(text) {
		return ComplexityComplex
	}

	return ComplexitySimple
}

// RouteModel selects the appropriate model based on query complexity.
// Returns the model name to use. If routing is disabled, returns empty string
// (caller should use the default model).
func RouteModel(text string, cfg RoutingConfig) string {
	if !cfg.Enabled {
		return ""
	}

	complexity := ClassifyQuery(text, cfg)

	switch complexity {
	case ComplexitySimple:
		if cfg.CheapModel != "" {
			return cfg.CheapModel
		}
	case ComplexityComplex:
		if cfg.CapableModel != "" {
			return cfg.CapableModel
		}
	}

	return ""
}

// countWords counts the number of whitespace-separated words in text.
func countWords(text string) int {
	count := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				count++
				inWord = false
			}
		} else {
			inWord = true
		}
	}
	if inWord {
		count++
	}
	return count
}

// containsCodeMarkers checks if text contains code-related markers.
func containsCodeMarkers(text string) bool {
	codeMarkers := []string{
		"```",
		"`",
		"func ",
		"function ",
		"def ",
		"class ",
		"import ",
		"package ",
		"const ",
		"var ",
		"let ",
		"if err != nil",
		"try {",
		"catch (",
		"#include",
		"fmt.Println",
		"console.log",
		"print(",
		"System.out",
	}

	lower := strings.ToLower(text)
	for _, marker := range codeMarkers {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

// containsComplexPatterns checks for patterns indicating a complex request.
func containsComplexPatterns(text string) bool {
	complexPatterns := []string{
		"explain",
		"analyze",
		"compare",
		"implement",
		"refactor",
		"debug",
		"optimize",
		"architecture",
		"design pattern",
		"algorithm",
		"write a ",
		"create a ",
		"build a ",
		"help me write",
		"step by step",
		"in detail",
		"comprehensive",
		"pros and cons",
		"trade-off",
		"tradeoff",
	}

	lower := strings.ToLower(text)
	for _, pattern := range complexPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
