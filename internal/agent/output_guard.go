package agent

import (
	"regexp"
	"strings"
)

// OutputGuard filters agent responses to prevent accidental leakage
// of sensitive information like API keys, internal paths, or config secrets.
type OutputGuard struct {
	patterns []*guardPattern
}

type guardPattern struct {
	re          *regexp.Regexp
	replacement string
	label       string
}

// NewOutputGuard creates an output guard with default sensitive patterns.
func NewOutputGuard() *OutputGuard {
	return &OutputGuard{
		patterns: defaultGuardPatterns,
	}
}

// Filter scans the response text and redacts any sensitive information found.
// Returns the filtered text and a list of matched pattern labels.
func (g *OutputGuard) Filter(text string) (string, []string) {
	var matched []string
	result := text

	for _, p := range g.patterns {
		if p.re.MatchString(result) {
			matched = append(matched, p.label)
			result = p.re.ReplaceAllString(result, p.replacement)
		}
	}

	// Also check for common config file content that shouldn't be in responses.
	if containsConfigLeak(result) {
		matched = append(matched, "config_leak")
		result = redactConfigBlock(result)
	}

	return result, matched
}

var defaultGuardPatterns = []*guardPattern{
	// API keys (various formats).
	{regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`), "[REDACTED_API_KEY]", "openai_key"},
	{regexp.MustCompile(`sk-ant-[a-zA-Z0-9\-]{20,}`), "[REDACTED_API_KEY]", "anthropic_key"},
	{regexp.MustCompile(`sk-or-[a-zA-Z0-9\-]{20,}`), "[REDACTED_API_KEY]", "openrouter_key"},
	{regexp.MustCompile(`AIza[a-zA-Z0-9\-_]{30,}`), "[REDACTED_API_KEY]", "google_key"},
	{regexp.MustCompile(`xoxb-[a-zA-Z0-9\-]{20,}`), "[REDACTED_TOKEN]", "slack_bot_token"},
	{regexp.MustCompile(`xapp-[a-zA-Z0-9\-]{20,}`), "[REDACTED_TOKEN]", "slack_app_token"},
	{regexp.MustCompile(`ghp_[a-zA-Z0-9]{30,}`), "[REDACTED_TOKEN]", "github_token"},
	{regexp.MustCompile(`gho_[a-zA-Z0-9]{30,}`), "[REDACTED_TOKEN]", "github_oauth"},
	{regexp.MustCompile(`glpat-[a-zA-Z0-9\-]{20,}`), "[REDACTED_TOKEN]", "gitlab_token"},

	// Bearer tokens in output.
	{regexp.MustCompile(`Bearer\s+[a-zA-Z0-9\-_.]{20,}`), "Bearer [REDACTED]", "bearer_token"},
	{regexp.MustCompile(`Authorization:\s*[a-zA-Z0-9\-_.]{20,}`), "Authorization: [REDACTED]", "auth_header"},

	// Environment variable assignments with sensitive names.
	{regexp.MustCompile(`(?i)(API_KEY|SECRET|TOKEN|PASSWORD|CREDENTIAL|AUTH_TOKEN)\s*=\s*[^\s]{8,}`), "$1=[REDACTED]", "env_secret"},

	// JWT tokens.
	{regexp.MustCompile(`eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+`), "[REDACTED_JWT]", "jwt_token"},

	// Private keys.
	{regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----[\s\S]*?-----END\s+(RSA\s+)?PRIVATE\s+KEY-----`), "[REDACTED_PRIVATE_KEY]", "private_key"},

	// Internal paths that could reveal system info.
	{regexp.MustCompile(`/Users/[a-zA-Z0-9]+/\.hera/\.env`), "[HERA_CONFIG_PATH]", "hera_env_path"},
}

// containsConfigLeak checks if the text contains what looks like dumped config.
func containsConfigLeak(text string) bool {
	indicators := []string{
		"api_key:",
		"COMPATIBLE_API_KEY=",
		"OPENAI_API_KEY=",
		"ANTHROPIC_API_KEY=",
		"base_url: https://",
		"jwt_secret:",
	}
	lower := strings.ToLower(text)
	count := 0
	for _, ind := range indicators {
		if strings.Contains(lower, strings.ToLower(ind)) {
			count++
		}
	}
	return count >= 2 // Two or more indicators = likely config dump.
}

// redactConfigBlock replaces config-like blocks with a notice.
func redactConfigBlock(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	redacting := false

	for _, line := range lines {
		lower := strings.ToLower(line)
		isSecret := strings.Contains(lower, "api_key") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "password")

		if isSecret && (strings.Contains(line, "=") || strings.Contains(line, ":")) {
			if !redacting {
				result = append(result, "[Configuration details redacted for security]")
				redacting = true
			}
			continue
		}
		redacting = false
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
