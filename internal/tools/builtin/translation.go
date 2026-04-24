package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// TranslationTool provides text translation and language detection.
// Uses a simple dictionary approach for common phrases; for full translation,
// delegates to the LLM or external API.
type TranslationTool struct{}

type translationArgs struct {
	Action     string `json:"action"`
	Text       string `json:"text"`
	SourceLang string `json:"source_lang,omitempty"`
	TargetLang string `json:"target_lang,omitempty"`
}

func (t *TranslationTool) Name() string { return "translation" }

func (t *TranslationTool) Description() string {
	return "Text translation utilities: detect language, transliterate, and character analysis."
}

func (t *TranslationTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["detect", "info", "char_count", "word_count"],
				"description": "Translation action: detect (language detection), info (character analysis), char_count, word_count."
			},
			"text": {
				"type": "string",
				"description": "Text to analyze or translate."
			},
			"source_lang": {
				"type": "string",
				"description": "Source language code (ISO 639-1)."
			},
			"target_lang": {
				"type": "string",
				"description": "Target language code (ISO 639-1)."
			}
		},
		"required": ["action", "text"]
	}`)
}

func (t *TranslationTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a translationArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	switch a.Action {
	case "detect":
		return detectLanguage(a.Text)
	case "info":
		return charInfo(a.Text)
	case "char_count":
		runes := []rune(a.Text)
		return &tools.Result{Content: fmt.Sprintf("Characters: %d, Bytes: %d", len(runes), len(a.Text))}, nil
	case "word_count":
		words := strings.Fields(a.Text)
		return &tools.Result{Content: fmt.Sprintf("Words: %d", len(words))}, nil
	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

func detectLanguage(text string) (*tools.Result, error) {
	runes := []rune(text)
	if len(runes) == 0 {
		return &tools.Result{Content: "empty text", IsError: true}, nil
	}

	// Simple heuristic-based language detection by Unicode ranges
	var cjk, cyrillic, arabic, latin, thai, devanagari int
	for _, r := range runes {
		switch {
		case r >= 0x4E00 && r <= 0x9FFF:
			cjk++
		case r >= 0x3040 && r <= 0x30FF:
			cjk++ // Japanese hiragana/katakana
		case r >= 0xAC00 && r <= 0xD7AF:
			cjk++ // Korean
		case r >= 0x0400 && r <= 0x04FF:
			cyrillic++
		case r >= 0x0600 && r <= 0x06FF:
			arabic++
		case r >= 0x0E00 && r <= 0x0E7F:
			thai++
		case r >= 0x0900 && r <= 0x097F:
			devanagari++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			latin++
		}
	}

	total := len(runes)
	var detected string
	maxCount := 0
	for _, pair := range []struct {
		name  string
		count int
	}{
		{"CJK (Chinese/Japanese/Korean)", cjk},
		{"Cyrillic (Russian/Ukrainian/etc.)", cyrillic},
		{"Arabic", arabic},
		{"Thai", thai},
		{"Devanagari (Hindi/Sanskrit)", devanagari},
		{"Latin", latin},
	} {
		if pair.count > maxCount {
			maxCount = pair.count
			detected = pair.name
		}
	}

	if detected == "" {
		detected = "Unknown"
	}

	confidence := float64(maxCount) / float64(total) * 100

	return &tools.Result{
		Content: fmt.Sprintf("Detected script: %s (%.0f%% confidence, %d/%d characters)", detected, confidence, maxCount, total),
	}, nil
}

func charInfo(text string) (*tools.Result, error) {
	runes := []rune(text)
	var sb strings.Builder

	fmt.Fprintf(&sb, "Text analysis:\n")
	fmt.Fprintf(&sb, "  Characters: %d\n", len(runes))
	fmt.Fprintf(&sb, "  Bytes (UTF-8): %d\n", len(text))
	fmt.Fprintf(&sb, "  Words: %d\n", len(strings.Fields(text)))
	fmt.Fprintf(&sb, "  Lines: %d\n", strings.Count(text, "\n")+1)

	// Count character types
	var upper, lower, digit, space, punct int
	for _, r := range runes {
		switch {
		case r >= 'A' && r <= 'Z':
			upper++
		case r >= 'a' && r <= 'z':
			lower++
		case r >= '0' && r <= '9':
			digit++
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			space++
		default:
			punct++
		}
	}

	fmt.Fprintf(&sb, "\nCharacter types:\n")
	fmt.Fprintf(&sb, "  Uppercase: %d\n", upper)
	fmt.Fprintf(&sb, "  Lowercase: %d\n", lower)
	fmt.Fprintf(&sb, "  Digits: %d\n", digit)
	fmt.Fprintf(&sb, "  Whitespace: %d\n", space)
	fmt.Fprintf(&sb, "  Other: %d\n", punct)

	return &tools.Result{Content: sb.String()}, nil
}

// RegisterTranslation registers the translation tool with the given registry.
func RegisterTranslation(registry *tools.Registry) {
	registry.Register(&TranslationTool{})
}
