package cli

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogEntry holds a single log entry captured by the ring buffer.
type LogEntry struct {
	Time    time.Time  `json:"time"`
	Level   slog.Level `json:"level"`
	Message string     `json:"message"`
}

// LogBuffer is a ring buffer that captures recent log entries for the /log tail
// command.
type LogBuffer struct {
	mu      sync.Mutex
	entries []LogEntry
	size    int
	pos     int
	count   int
}

// NewLogBuffer creates a ring buffer that holds up to size entries.
func NewLogBuffer(size int) *LogBuffer {
	if size <= 0 {
		size = 500
	}
	return &LogBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

// Append adds an entry to the buffer, overwriting the oldest if full.
func (lb *LogBuffer) Append(entry LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.entries[lb.pos] = entry
	lb.pos = (lb.pos + 1) % lb.size
	if lb.count < lb.size {
		lb.count++
	}
}

// Tail returns the last n entries (or all if n <= 0).
func (lb *LogBuffer) Tail(n int) []LogEntry {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if n <= 0 || n > lb.count {
		n = lb.count
	}

	result := make([]LogEntry, n)
	start := (lb.pos - n + lb.size) % lb.size
	for i := 0; i < n; i++ {
		result[i] = lb.entries[(start+i)%lb.size]
	}
	return result
}

// Clear removes all entries.
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.pos = 0
	lb.count = 0
}

// Count returns the number of stored entries.
func (lb *LogBuffer) Count() int {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.count
}

// logLevel is a package-level variable that can be adjusted at runtime.
var (
	logLevelMu   sync.Mutex
	logLevelVar  = new(slog.LevelVar)
	logBufferVar *LogBuffer
)

func init() {
	logLevelVar.Set(slog.LevelInfo)
}

// SetLogLevel changes the slog level at runtime.
func SetLogLevel(level slog.Level) {
	logLevelMu.Lock()
	defer logLevelMu.Unlock()
	logLevelVar.Set(level)
}

// GetLogLevel returns the current log level.
func GetLogLevel() slog.Level {
	logLevelMu.Lock()
	defer logLevelMu.Unlock()
	return logLevelVar.Level()
}

// LogLevelVar returns the shared LevelVar for use with slog handlers.
func LogLevelVar() *slog.LevelVar {
	return logLevelVar
}

// SetLogBuffer sets the global log buffer.
func SetLogBuffer(lb *LogBuffer) {
	logLevelMu.Lock()
	defer logLevelMu.Unlock()
	logBufferVar = lb
}

// GetLogBuffer returns the global log buffer.
func GetLogBuffer() *LogBuffer {
	logLevelMu.Lock()
	defer logLevelMu.Unlock()
	return logBufferVar
}

// HandleLogCommand processes /log subcommands.
func HandleLogCommand(args string) (string, error) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "Usage: /log level <debug|info|warn|error> | /log tail [n] | /log clear", nil
	}

	switch parts[0] {
	case "level":
		if len(parts) < 2 {
			return fmt.Sprintf("Current log level: %s", GetLogLevel().String()), nil
		}
		var level slog.Level
		switch strings.ToLower(parts[1]) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			return fmt.Sprintf("Unknown level: %s (use debug, info, warn, error)", parts[1]), nil
		}
		SetLogLevel(level)
		return fmt.Sprintf("Log level set to %s", level.String()), nil

	case "tail":
		buf := GetLogBuffer()
		if buf == nil {
			return "Log buffer not initialized.", nil
		}
		n := 20
		if len(parts) >= 2 {
			if parsed, err := strconv.Atoi(parts[1]); err == nil && parsed > 0 {
				n = parsed
			}
		}
		entries := buf.Tail(n)
		if len(entries) == 0 {
			return "No log entries.", nil
		}
		var sb strings.Builder
		for _, e := range entries {
			sb.WriteString(fmt.Sprintf("[%s] %s: %s\n",
				e.Time.Format("15:04:05"),
				e.Level.String(),
				e.Message,
			))
		}
		return sb.String(), nil

	case "clear":
		buf := GetLogBuffer()
		if buf == nil {
			return "Log buffer not initialized.", nil
		}
		buf.Clear()
		return "Log buffer cleared.", nil

	default:
		return "Usage: /log level <debug|info|warn|error> | /log tail [n] | /log clear", nil
	}
}
