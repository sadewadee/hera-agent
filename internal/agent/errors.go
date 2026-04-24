package agent

import (
	"math"
	"net/http"
	"strings"
	"time"
)

// ErrorCategory classifies LLM/API errors for retry strategy decisions.
type ErrorCategory int

const (
	// ErrorTransient represents temporary network or server glitches.
	ErrorTransient ErrorCategory = iota
	// ErrorRateLimit represents rate limiting (429).
	ErrorRateLimit
	// ErrorAuth represents authentication/authorization failures.
	ErrorAuth
	// ErrorInvalidRequest represents malformed or invalid requests.
	ErrorInvalidRequest
	// ErrorContextOverflow represents context length exceeded errors.
	ErrorContextOverflow
	// ErrorServerError represents server-side errors (500+).
	ErrorServerError
	// ErrorUnknown represents unclassifiable errors.
	ErrorUnknown
)

// String returns a human-readable name for the error category.
func (c ErrorCategory) String() string {
	switch c {
	case ErrorTransient:
		return "transient"
	case ErrorRateLimit:
		return "rate_limit"
	case ErrorAuth:
		return "auth"
	case ErrorInvalidRequest:
		return "invalid_request"
	case ErrorContextOverflow:
		return "context_overflow"
	case ErrorServerError:
		return "server_error"
	default:
		return "unknown"
	}
}

// ClassifyError categorizes an error from an LLM/API call based on the error
// message content and common patterns across providers.
func ClassifyError(err error) ErrorCategory {
	if err == nil {
		return ErrorUnknown
	}

	msg := strings.ToLower(err.Error())

	// Rate limiting indicators.
	if strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "quota exceeded") ||
		strings.Contains(msg, "throttl") {
		return ErrorRateLimit
	}

	// Authentication/authorization.
	if strings.Contains(msg, "401") ||
		strings.Contains(msg, "403") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "invalid api key") ||
		strings.Contains(msg, "invalid_api_key") ||
		strings.Contains(msg, "authentication") ||
		strings.Contains(msg, "permission denied") {
		return ErrorAuth
	}

	// Context overflow.
	if strings.Contains(msg, "context length") ||
		strings.Contains(msg, "context_length_exceeded") ||
		strings.Contains(msg, "max_tokens") ||
		strings.Contains(msg, "token limit") ||
		strings.Contains(msg, "maximum context") ||
		strings.Contains(msg, "too many tokens") ||
		strings.Contains(msg, "input too long") {
		return ErrorContextOverflow
	}

	// Invalid request.
	if strings.Contains(msg, "400") ||
		strings.Contains(msg, "bad request") ||
		strings.Contains(msg, "invalid_request") ||
		strings.Contains(msg, "invalid request") ||
		strings.Contains(msg, "invalid model") ||
		strings.Contains(msg, "malformed") ||
		strings.Contains(msg, "validation error") {
		return ErrorInvalidRequest
	}

	// Server errors (5xx).
	if strings.Contains(msg, "500") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "internal server error") ||
		strings.Contains(msg, "bad gateway") ||
		strings.Contains(msg, "service unavailable") ||
		strings.Contains(msg, "gateway timeout") ||
		strings.Contains(msg, "server error") {
		return ErrorServerError
	}

	// Transient / network errors.
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "temporary") ||
		strings.Contains(msg, "network") {
		return ErrorTransient
	}

	return ErrorUnknown
}

// ClassifyHTTPStatus categorizes an error based on HTTP status code.
func ClassifyHTTPStatus(statusCode int) ErrorCategory {
	switch {
	case statusCode == http.StatusTooManyRequests:
		return ErrorRateLimit
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ErrorAuth
	case statusCode == http.StatusBadRequest:
		return ErrorInvalidRequest
	case statusCode >= 500:
		return ErrorServerError
	case statusCode == http.StatusRequestEntityTooLarge:
		return ErrorContextOverflow
	default:
		return ErrorUnknown
	}
}

// ShouldRetry returns true if the error category warrants a retry attempt.
func ShouldRetry(category ErrorCategory) bool {
	switch category {
	case ErrorTransient, ErrorRateLimit, ErrorServerError:
		return true
	case ErrorAuth, ErrorInvalidRequest, ErrorContextOverflow:
		return false
	default:
		return false
	}
}

// RetryDelay returns the recommended delay before the next retry attempt.
// Uses exponential backoff for transient errors and longer delays for rate limits.
func RetryDelay(category ErrorCategory, attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	switch category {
	case ErrorRateLimit:
		// Rate limits: start at 5s, double each attempt, cap at 120s.
		base := 5.0
		delay := base * math.Pow(2, float64(attempt))
		if delay > 120 {
			delay = 120
		}
		return time.Duration(delay) * time.Second

	case ErrorTransient:
		// Transient: start at 1s, double each attempt, cap at 30s.
		base := 1.0
		delay := base * math.Pow(2, float64(attempt))
		if delay > 30 {
			delay = 30
		}
		return time.Duration(delay) * time.Second

	case ErrorServerError:
		// Server errors: start at 2s, double each attempt, cap at 60s.
		base := 2.0
		delay := base * math.Pow(2, float64(attempt))
		if delay > 60 {
			delay = 60
		}
		return time.Duration(delay) * time.Second

	default:
		return 0
	}
}
