package hcore

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(level string, jsonFormat bool) *slog.Logger {
	logger, _ := SetupLogging(LogConfig{Level: level, JSONFormat: jsonFormat})
	return logger
}

func TestSetupLogging_Info(t *testing.T) {
	logger := setup("info", false)
	assert.NotNil(t, logger)
	assert.True(t, logger.Enabled(nil, slog.LevelInfo))
	assert.False(t, logger.Enabled(nil, slog.LevelDebug))
}

func TestSetupLogging_Debug(t *testing.T) {
	logger := setup("debug", false)
	assert.NotNil(t, logger)
	assert.True(t, logger.Enabled(nil, slog.LevelDebug))
}

func TestSetupLogging_Warn(t *testing.T) {
	logger := setup("warn", false)
	assert.NotNil(t, logger)
	assert.True(t, logger.Enabled(nil, slog.LevelWarn))
	assert.False(t, logger.Enabled(nil, slog.LevelInfo))
}

func TestSetupLogging_Error(t *testing.T) {
	logger := setup("error", false)
	assert.NotNil(t, logger)
	assert.True(t, logger.Enabled(nil, slog.LevelError))
	assert.False(t, logger.Enabled(nil, slog.LevelWarn))
}

func TestSetupLogging_Unknown(t *testing.T) {
	logger := setup("unknown", false)
	assert.NotNil(t, logger)
	assert.True(t, logger.Enabled(nil, slog.LevelInfo))
}

func TestSetupLogging_JSONFormat(t *testing.T) {
	logger := setup("info", true)
	assert.NotNil(t, logger)
}

func TestSetupLogging_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	logger, cleanup := SetupLogging(LogConfig{Level: "info", LogDir: dir})
	defer cleanup()
	require.NotNil(t, logger)

	logger.Info("hello from test", "key", "value")

	// slog is buffered at the handler level for TextHandler; reading the
	// underlying file immediately after a single log line should still see it
	// because slog.Handler flushes per record.
	data, err := os.ReadFile(filepath.Join(dir, "agent.log"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "hello from test")
	assert.Contains(t, string(data), "key=value")
}

func TestSetupLogging_RotatesWhenSizeExceeded(t *testing.T) {
	dir := t.TempDir()
	logger, cleanup := SetupLogging(LogConfig{
		Level:    "info",
		LogDir:   dir,
		MaxBytes: 256, // small, forces rotation quickly
		MaxFiles: 3,
	})
	defer cleanup()

	payload := strings.Repeat("x", 100)
	for i := 0; i < 20; i++ {
		logger.Info("rotate-test", "payload", payload)
	}

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var logFiles []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "agent.log") {
			logFiles = append(logFiles, e.Name())
		}
	}
	// Expect agent.log + at least agent.log.1 after rotation.
	assert.GreaterOrEqual(t, len(logFiles), 2, "expected rotation to produce multiple files, got %v", logFiles)
	// Never exceeds MaxFiles (current + rotated .1 .. .MaxFiles-1 = MaxFiles total).
	assert.LessOrEqual(t, len(logFiles), 3)
}
