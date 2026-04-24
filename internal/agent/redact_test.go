package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPIIRedactor(t *testing.T) {
	r := NewPIIRedactor(true)
	require.NotNil(t, r)
	assert.True(t, r.enabled)
	assert.NotEmpty(t, r.patterns)
}

func TestPIIRedactor_Redact_Email(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Contact me at john@example.com for details"
	result, count := r.Redact(text)
	assert.Contains(t, result, "[EMAIL_REDACTED]")
	assert.NotContains(t, result, "john@example.com")
	assert.Equal(t, 1, count)
}

func TestPIIRedactor_Redact_SSN(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "My SSN is 123-45-6789"
	result, count := r.Redact(text)
	assert.Contains(t, result, "[SSN_REDACTED]")
	assert.NotContains(t, result, "123-45-6789")
	assert.GreaterOrEqual(t, count, 1)
}

func TestPIIRedactor_Redact_Phone(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Call me at (555) 123-4567"
	result, count := r.Redact(text)
	assert.Contains(t, result, "[PHONE_REDACTED]")
	assert.GreaterOrEqual(t, count, 1)
}

func TestPIIRedactor_Redact_IPAddress(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Server at 192.168.1.100"
	result, count := r.Redact(text)
	assert.Contains(t, result, "[IP_REDACTED]")
	assert.GreaterOrEqual(t, count, 1)
}

func TestPIIRedactor_Redact_Multiple(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Email: john@test.com, IP: 10.0.0.1"
	result, count := r.Redact(text)
	assert.Contains(t, result, "[EMAIL_REDACTED]")
	assert.Contains(t, result, "[IP_REDACTED]")
	assert.GreaterOrEqual(t, count, 2)
}

func TestPIIRedactor_Redact_NoPII(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Hello world, this is a clean text"
	result, count := r.Redact(text)
	assert.Equal(t, text, result)
	assert.Equal(t, 0, count)
}

func TestPIIRedactor_Redact_Disabled(t *testing.T) {
	r := NewPIIRedactor(false)
	text := "Contact john@example.com"
	result, count := r.Redact(text)
	assert.Equal(t, text, result)
	assert.Equal(t, 0, count)
}

func TestPIIRedactor_DetectPII(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Email: john@test.com, SSN: 123-45-6789"
	types := r.DetectPII(text)
	assert.Contains(t, types, PIIEmail)
	assert.Contains(t, types, PIISSN)
}

func TestPIIRedactor_DetectPII_None(t *testing.T) {
	r := NewPIIRedactor(true)
	types := r.DetectPII("clean text with no PII")
	assert.Empty(t, types)
}

func TestPIIRedactor_RedactForLogging(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Key is sk-abc123456789012345678901234567 and user is john@test.com"
	result := r.RedactForLogging(text)
	assert.Contains(t, result, "[API_KEY_REDACTED]")
	assert.Contains(t, result, "[EMAIL_REDACTED]")
	assert.NotContains(t, result, "sk-abc")
	assert.NotContains(t, result, "john@test.com")
}

func TestPIIRedactor_RedactForLogging_BearerToken(t *testing.T) {
	r := NewPIIRedactor(true)
	text := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.test"
	result := r.RedactForLogging(text)
	assert.Contains(t, result, "Bearer [TOKEN_REDACTED]")
}

func TestPIIRedactor_RedactForLogging_Disabled(t *testing.T) {
	r := NewPIIRedactor(false)
	text := "john@test.com with sk-secret"
	result := r.RedactForLogging(text)
	assert.Equal(t, text, result)
}

func TestContainsPII(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"email", "Contact john@test.com", true},
		{"phone", "Call 555-123-4567", true},
		{"ip", "IP is 192.168.0.1", true},
		{"clean", "Hello world", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ContainsPII(tt.text))
		})
	}
}
