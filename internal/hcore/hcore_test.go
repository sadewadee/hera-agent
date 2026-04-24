package hcore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Constants ---

func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0.0.142", Version)
	assert.Equal(t, "hera", AppName)
	assert.Equal(t, 8080, DefaultPort)
	assert.Equal(t, 100*1024, MaxMessageSize)
	assert.Equal(t, 10*1024*1024, MaxFileSize)
	assert.Equal(t, "hera.db", DefaultDBName)
	assert.Equal(t, "hera.yaml", ConfigFileName)
	assert.Equal(t, "SOUL.md", SOULFileName)
	assert.Equal(t, ".hera.md", HintFileName)
	assert.Equal(t, "gpt-4o", DefaultModel)
	assert.Equal(t, 120, DefaultTimeout)
}

func TestDefaultSkillDirs(t *testing.T) {
	t.Parallel()

	assert.Len(t, DefaultSkillDirs, 2)
	assert.Contains(t, DefaultSkillDirs, "skills")
	assert.Contains(t, DefaultSkillDirs, "optional-skills")
}

func TestSupportedPlatforms(t *testing.T) {
	t.Parallel()

	assert.Greater(t, len(SupportedPlatforms), 0)
	assert.Contains(t, SupportedPlatforms, "cli")
	assert.Contains(t, SupportedPlatforms, "telegram")
	assert.Contains(t, SupportedPlatforms, "discord")
	assert.Contains(t, SupportedPlatforms, "slack")
}

// --- HumanDuration ---

func TestHumanDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{name: "milliseconds", d: 500 * time.Millisecond, want: "500ms"},
		{name: "zero", d: 0, want: "0ms"},
		{name: "seconds", d: 30 * time.Second, want: "30s"},
		{name: "one second", d: time.Second, want: "1s"},
		{name: "minutes and seconds", d: 2*time.Minute + 30*time.Second, want: "2m30s"},
		{name: "hours and minutes", d: 3*time.Hour + 15*time.Minute, want: "3h15m"},
		{name: "days and hours", d: 50 * time.Hour, want: "2d2h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HumanDuration(tt.d))
		})
	}
}

// --- RelativeTime ---

func TestRelativeTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{name: "just now", ago: 10 * time.Second, want: "just now"},
		{name: "minutes ago", ago: 5 * time.Minute, want: "5 minutes ago"},
		{name: "hours ago", ago: 3 * time.Hour, want: "3 hours ago"},
		{name: "days ago", ago: 48 * time.Hour, want: "2 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RelativeTime(time.Now().Add(-tt.ago))
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ParseDuration ---

func TestParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{name: "days", input: "3d", want: 3 * 24 * time.Hour},
		{name: "weeks", input: "2w", want: 2 * 7 * 24 * time.Hour},
		{name: "standard hours", input: "2h", want: 2 * time.Hour},
		{name: "standard minutes", input: "30m", want: 30 * time.Minute},
		{name: "standard seconds", input: "10s", want: 10 * time.Second},
		{name: "one day", input: "1d", want: 24 * time.Hour},
		{name: "one week", input: "1w", want: 7 * 24 * time.Hour},
		{name: "invalid", input: "abc", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// --- SetupLogging ---

func TestSetupLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		level      string
		jsonFormat bool
	}{
		{name: "info text", level: "info", jsonFormat: false},
		{name: "debug json", level: "debug", jsonFormat: true},
		{name: "warn text", level: "warn", jsonFormat: false},
		{name: "error json", level: "error", jsonFormat: true},
		{name: "default level", level: "unknown", jsonFormat: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := SetupLogging(LogConfig{Level: tt.level, JSONFormat: tt.jsonFormat})
			assert.NotNil(t, logger)
		})
	}
}

// --- AppState ---

func TestNewAppState(t *testing.T) {
	t.Parallel()

	state := NewAppState()
	require.NotNil(t, state)
	assert.True(t, state.IsRunning())
}

func TestAppStateShutdown(t *testing.T) {
	t.Parallel()

	state := NewAppState()
	assert.True(t, state.IsRunning())

	state.Shutdown()
	assert.False(t, state.IsRunning())
}

func TestAppStateShutdownIdempotent(t *testing.T) {
	t.Parallel()

	state := NewAppState()
	state.Shutdown()
	state.Shutdown() // second call should not panic
	assert.False(t, state.IsRunning())
}

func TestAppStateDone(t *testing.T) {
	t.Parallel()

	state := NewAppState()
	done := state.Done()
	require.NotNil(t, done)

	// Channel should not be closed yet.
	select {
	case <-done:
		t.Fatal("done channel should not be closed before shutdown")
	default:
		// expected
	}

	state.Shutdown()

	// Channel should be closed now.
	select {
	case <-done:
		// expected
	default:
		t.Fatal("done channel should be closed after shutdown")
	}
}
