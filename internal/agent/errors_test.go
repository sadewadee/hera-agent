package agent

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		cat    ErrorCategory
		expect string
	}{
		{ErrorTransient, "transient"},
		{ErrorRateLimit, "rate_limit"},
		{ErrorAuth, "auth"},
		{ErrorInvalidRequest, "invalid_request"},
		{ErrorContextOverflow, "context_overflow"},
		{ErrorServerError, "server_error"},
		{ErrorUnknown, "unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expect, tt.cat.String())
	}
}

func TestClassifyError_NilError(t *testing.T) {
	assert.Equal(t, ErrorUnknown, ClassifyError(nil))
}

func TestClassifyError_RateLimit(t *testing.T) {
	tests := []string{
		"HTTP 429: too many requests",
		"rate limit exceeded",
		"quota exceeded for model",
		"request throttled",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, ErrorRateLimit, ClassifyError(errors.New(msg)))
		})
	}
}

func TestClassifyError_Auth(t *testing.T) {
	tests := []string{
		"HTTP 401 unauthorized",
		"HTTP 403 forbidden",
		"invalid api key provided",
		"invalid_api_key",
		"authentication failed",
		"permission denied",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, ErrorAuth, ClassifyError(errors.New(msg)))
		})
	}
}

func TestClassifyError_ContextOverflow(t *testing.T) {
	tests := []string{
		"context length exceeded",
		"context_length_exceeded",
		"max_tokens exceeded",
		"token limit reached",
		"maximum context window",
		"too many tokens",
		"input too long",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, ErrorContextOverflow, ClassifyError(errors.New(msg)))
		})
	}
}

func TestClassifyError_InvalidRequest(t *testing.T) {
	tests := []string{
		"HTTP 400 bad request",
		"invalid_request_error",
		"invalid model specified",
		"malformed JSON",
		"validation error in field",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, ErrorInvalidRequest, ClassifyError(errors.New(msg)))
		})
	}
}

func TestClassifyError_ServerError(t *testing.T) {
	tests := []string{
		"HTTP 500 internal server error",
		"HTTP 502 bad gateway",
		"HTTP 503 service unavailable",
		"HTTP 504 gateway timeout",
		"server error occurred",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, ErrorServerError, ClassifyError(errors.New(msg)))
		})
	}
}

func TestClassifyError_Transient(t *testing.T) {
	tests := []string{
		"connection timeout",
		"request timed out",
		"connection reset by peer",
		"connection refused",
		"unexpected eof",
		"broken pipe",
		"no such host",
		"temporary failure",
		"network unreachable",
	}
	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, ErrorTransient, ClassifyError(errors.New(msg)))
		})
	}
}

func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		status int
		expect ErrorCategory
	}{
		{http.StatusTooManyRequests, ErrorRateLimit},
		{http.StatusUnauthorized, ErrorAuth},
		{http.StatusForbidden, ErrorAuth},
		{http.StatusBadRequest, ErrorInvalidRequest},
		{http.StatusInternalServerError, ErrorServerError},
		{http.StatusBadGateway, ErrorServerError},
		{http.StatusServiceUnavailable, ErrorServerError},
		{http.StatusRequestEntityTooLarge, ErrorContextOverflow},
		{http.StatusOK, ErrorUnknown},
		{http.StatusNotFound, ErrorUnknown},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expect, ClassifyHTTPStatus(tt.status))
		})
	}
}

func TestShouldRetry(t *testing.T) {
	assert.True(t, ShouldRetry(ErrorTransient))
	assert.True(t, ShouldRetry(ErrorRateLimit))
	assert.True(t, ShouldRetry(ErrorServerError))
	assert.False(t, ShouldRetry(ErrorAuth))
	assert.False(t, ShouldRetry(ErrorInvalidRequest))
	assert.False(t, ShouldRetry(ErrorContextOverflow))
	assert.False(t, ShouldRetry(ErrorUnknown))
}

func TestRetryDelay_RateLimit(t *testing.T) {
	d0 := RetryDelay(ErrorRateLimit, 0)
	d1 := RetryDelay(ErrorRateLimit, 1)
	assert.Equal(t, 5*time.Second, d0)
	assert.Equal(t, 10*time.Second, d1)
}

func TestRetryDelay_Transient(t *testing.T) {
	d0 := RetryDelay(ErrorTransient, 0)
	d1 := RetryDelay(ErrorTransient, 1)
	assert.Equal(t, 1*time.Second, d0)
	assert.Equal(t, 2*time.Second, d1)
}

func TestRetryDelay_ServerError(t *testing.T) {
	d0 := RetryDelay(ErrorServerError, 0)
	assert.Equal(t, 2*time.Second, d0)
}

func TestRetryDelay_CapsAtMax(t *testing.T) {
	d := RetryDelay(ErrorRateLimit, 100)
	assert.Equal(t, 120*time.Second, d)

	d = RetryDelay(ErrorTransient, 100)
	assert.Equal(t, 30*time.Second, d)

	d = RetryDelay(ErrorServerError, 100)
	assert.Equal(t, 60*time.Second, d)
}

func TestRetryDelay_NonRetryable(t *testing.T) {
	assert.Equal(t, time.Duration(0), RetryDelay(ErrorAuth, 0))
	assert.Equal(t, time.Duration(0), RetryDelay(ErrorInvalidRequest, 0))
}

func TestRetryDelay_NegativeAttempt(t *testing.T) {
	d := RetryDelay(ErrorRateLimit, -5)
	assert.Equal(t, 5*time.Second, d) // treated as attempt 0
}
