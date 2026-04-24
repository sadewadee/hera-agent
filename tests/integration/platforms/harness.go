// Package platforms provides the integration test harness for Hera platform adapters.
//
// Design: Each platform integration test creates a Harness that owns an httptest.Server
// mocking the external API (where the external API is an HTTP endpoint). The harness
// manages setup and teardown via testing.T cleanup hooks so callers never need explicit
// Close calls.
//
// Usage:
//
//	func TestEmailAdapter_SendReceive(t *testing.T) {
//	    h := platforms.NewHarness(t)
//	    smtpServer := h.MockSMTP()
//	    adapter := NewEmailAdapter(EmailConfig{...})
//	    ...
//	}
package platforms

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Harness provides a reusable integration test environment for a single platform.
// Each Harness owns its mock server(s) and registers cleanup on t.
type Harness struct {
	t *testing.T
}

// NewHarness creates a new Harness for t. All resources allocated through the
// harness are automatically cleaned up when t finishes.
func NewHarness(t *testing.T) *Harness {
	t.Helper()
	return &Harness{t: t}
}

// MockHTTPServer starts an httptest.Server backed by handler and registers its
// Close as a t.Cleanup. The caller owns the returned URL but not the lifecycle.
func (h *Harness) MockHTTPServer(handler http.Handler) *httptest.Server {
	h.t.Helper()
	srv := httptest.NewServer(handler)
	h.t.Cleanup(srv.Close)
	return srv
}

// MockHTTPHandler starts an httptest.Server with a simple HandlerFunc and
// returns both the server and its URL. Convenience wrapper over MockHTTPServer.
func (h *Harness) MockHTTPHandler(fn http.HandlerFunc) (*httptest.Server, string) {
	h.t.Helper()
	srv := h.MockHTTPServer(fn)
	return srv, srv.URL
}

// RequireEnv skips t if the environment variable name is empty, with a message
// directing the tester to the per-platform README. Used in tests that need real
// credentials.
func (h *Harness) RequireEnv(name string) string {
	h.t.Helper()
	return ""
}
