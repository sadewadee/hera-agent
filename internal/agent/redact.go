package agent

import (
	"regexp"
	"strings"
)

// PIIType represents a category of personally identifiable information.
type PIIType string

const (
	PIIEmail      PIIType = "email"
	PIIPhone      PIIType = "phone"
	PIISSN        PIIType = "ssn"
	PIICreditCard PIIType = "credit_card"
	PIIIPAddress  PIIType = "ip_address"
)

// piiPattern pairs a PII type with its regex pattern.
type piiPattern struct {
	Type    PIIType
	Pattern *regexp.Regexp
	Replace string
}

// PIIRedactor scans and redacts personally identifiable information from text.
type PIIRedactor struct {
	patterns []piiPattern
	enabled  bool
}

// NewPIIRedactor creates a new PII redactor. If enabled is false, Redact is a no-op.
func NewPIIRedactor(enabled bool) *PIIRedactor {
	r := &PIIRedactor{
		enabled: enabled,
		patterns: []piiPattern{
			{
				Type:    PIIEmail,
				Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
				Replace: "[EMAIL_REDACTED]",
			},
			{
				Type:    PIISSN,
				Pattern: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
				Replace: "[SSN_REDACTED]",
			},
			{
				Type:    PIICreditCard,
				Pattern: regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
				Replace: "[CREDIT_CARD_REDACTED]",
			},
			{
				Type:    PIIPhone,
				Pattern: regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`),
				Replace: "[PHONE_REDACTED]",
			},
			{
				Type:    PIIIPAddress,
				Pattern: regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
				Replace: "[IP_REDACTED]",
			},
		},
	}
	return r
}

// Redact replaces all detected PII in text with redaction placeholders.
// Returns the redacted text and the count of redactions made.
func (r *PIIRedactor) Redact(text string) (string, int) {
	if !r.enabled {
		return text, 0
	}

	count := 0
	result := text
	for _, p := range r.patterns {
		matches := p.Pattern.FindAllStringIndex(result, -1)
		if len(matches) > 0 {
			count += len(matches)
			result = p.Pattern.ReplaceAllString(result, p.Replace)
		}
	}
	return result, count
}

// DetectPII scans text and returns all detected PII types found.
func (r *PIIRedactor) DetectPII(text string) []PIIType {
	var found []PIIType
	seen := make(map[PIIType]bool)

	for _, p := range r.patterns {
		if p.Pattern.MatchString(text) && !seen[p.Type] {
			found = append(found, p.Type)
			seen[p.Type] = true
		}
	}
	return found
}

// RedactForLogging applies redaction specifically for log output.
// More aggressive: also redacts potential API keys and tokens.
func (r *PIIRedactor) RedactForLogging(text string) string {
	if !r.enabled {
		return text
	}

	result, _ := r.Redact(text)

	// Redact potential API keys (long alphanumeric strings that look like keys).
	apiKeyPattern := regexp.MustCompile(`\b(sk-[a-zA-Z0-9]{20,})\b`)
	result = apiKeyPattern.ReplaceAllString(result, "[API_KEY_REDACTED]")

	// Redact Bearer tokens.
	bearerPattern := regexp.MustCompile(`Bearer\s+[a-zA-Z0-9._\-]+`)
	result = bearerPattern.ReplaceAllString(result, "Bearer [TOKEN_REDACTED]")

	return result
}

// ContainsPII returns true if the text contains any detectable PII.
func ContainsPII(text string) bool {
	// Quick check using simple string patterns before regex.
	if strings.Contains(text, "@") ||
		strings.Contains(text, "xxx-xx-") ||
		strings.ContainsAny(text, "0123456789") {
		r := NewPIIRedactor(true)
		return len(r.DetectPII(text)) > 0
	}
	return false
}
