package platforms

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// CLIAdapter reads from stdin and writes to stdout, implementing the
// PlatformAdapter interface for local terminal interaction.
type CLIAdapter struct {
	BaseAdapter
	reader   io.Reader
	writer   io.Writer
	username string
	done     chan struct{}
}

// CLIOption configures the CLI adapter.
type CLIOption func(*CLIAdapter)

// WithCLIReader overrides the input source (default: os.Stdin).
func WithCLIReader(r io.Reader) CLIOption {
	return func(c *CLIAdapter) { c.reader = r }
}

// WithCLIWriter overrides the output destination (default: os.Stdout).
func WithCLIWriter(w io.Writer) CLIOption {
	return func(c *CLIAdapter) { c.writer = w }
}

// WithCLIUsername sets the local user identity.
func WithCLIUsername(name string) CLIOption {
	return func(c *CLIAdapter) { c.username = name }
}

// NewCLIAdapter creates a CLI platform adapter.
func NewCLIAdapter(opts ...CLIOption) *CLIAdapter {
	c := &CLIAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "cli"},
		reader:      os.Stdin,
		writer:      os.Stdout,
		username:    "local",
		done:        make(chan struct{}),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Connect starts the input scanning loop. It blocks by running the
// read loop in a goroutine and returns immediately.
func (c *CLIAdapter) Connect(ctx context.Context) error {
	c.SetConnected(true)
	go c.readLoop(ctx)
	return nil
}

// Disconnect signals the read loop to stop.
func (c *CLIAdapter) Disconnect(_ context.Context) error {
	c.SetConnected(false)
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	return nil
}

// Send writes a plain-text message to the output writer. Markdown formatting
// is always stripped for the CLI since a terminal cannot render it.
func (c *CLIAdapter) Send(_ context.Context, _ string, msg gateway.OutgoingMessage) error {
	_, err := fmt.Fprintln(c.writer, StripMarkdown(msg.Text))
	return err
}

// GetChatInfo returns information about the CLI "chat".
func (c *CLIAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       "cli",
		Title:    "Terminal",
		Type:     "private",
		Platform: "cli",
	}, nil
}

// SupportedMedia returns the media types supported by the CLI adapter.
// CLI only supports text; media types are listed for compatibility but
// will be rendered as text references.
func (c *CLIAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{}
}

// readLoop scans stdin line by line and dispatches messages to the handler.
func (c *CLIAdapter) readLoop(ctx context.Context) {
	scanner := bufio.NewScanner(c.reader)
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
		}

		if !scanner.Scan() {
			// EOF or error -- disconnect gracefully.
			c.SetConnected(false)
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		h := c.Handler()
		if h != nil {
			msg := gateway.IncomingMessage{
				Platform:  "cli",
				ChatID:    "cli",
				UserID:    c.username,
				Username:  c.username,
				Text:      line,
				Timestamp: time.Now(),
			}
			h(ctx, msg)
		}
	}
}
