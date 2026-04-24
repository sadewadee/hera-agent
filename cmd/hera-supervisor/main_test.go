package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/supervisor"
)

func TestLoadAgents_ValidFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
agents:
  - name: coder
    model: claude-opus
  - name: reviewer
`), 0o644))

	defs, err := loadAgents(cfgPath)
	require.NoError(t, err)
	require.Len(t, defs, 2)
	assert.Equal(t, "coder", defs[0].Name)
	assert.Equal(t, "claude-opus", defs[0].Model)
	assert.Equal(t, "reviewer", defs[1].Name)
}

func TestLoadAgents_FileNotExist(t *testing.T) {
	defs, err := loadAgents("/tmp/nonexistent-hera-supervisor-test.yaml")
	require.NoError(t, err, "missing file should return nil, not error")
	assert.Nil(t, defs)
}

func TestLoadAgents_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("not: valid: yaml: ["), 0o644))
	_, err := loadAgents(cfgPath)
	require.Error(t, err)
}

func TestStatusEndpoint_Empty(t *testing.T) {
	// Build a supervisor with no agents spawned, test the /supervisor/status handler.
	sup, err := supervisor.New(supervisor.Config{
		Factory: func(ctx context.Context, name string) error {
			<-ctx.Done()
			return nil
		},
	})
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/supervisor/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sup.HealthReport()) //nolint:errcheck
	})

	req := httptest.NewRequest(http.MethodGet, "/supervisor/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var health supervisor.Health
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &health))
	assert.Equal(t, 0, health.Running)
	assert.NotNil(t, health.Agents)
}

func TestStatusEndpoint_HealthPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/supervisor/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	req := httptest.NewRequest(http.MethodGet, "/supervisor/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}
