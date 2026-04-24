package platforms

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestNewCLIAdapter(t *testing.T) {
	a := NewCLIAdapter()
	require.NotNil(t, a)
	assert.Equal(t, "cli", a.Name())
	assert.False(t, a.IsConnected())
}

func TestNewCLIAdapter_WithOptions(t *testing.T) {
	r := strings.NewReader("input")
	w := &bytes.Buffer{}
	a := NewCLIAdapter(
		WithCLIReader(r),
		WithCLIWriter(w),
		WithCLIUsername("alice"),
	)
	assert.Equal(t, "alice", a.username)
}

func TestCLIAdapter_ConnectDisconnect(t *testing.T) {
	r := strings.NewReader("")
	a := NewCLIAdapter(WithCLIReader(r))

	err := a.Connect(t.Context())
	require.NoError(t, err)
	assert.True(t, a.IsConnected())

	err = a.Disconnect(t.Context())
	require.NoError(t, err)
	assert.False(t, a.IsConnected())
}

func TestCLIAdapter_Send(t *testing.T) {
	var buf bytes.Buffer
	a := NewCLIAdapter(WithCLIWriter(&buf))

	err := a.Send(t.Context(), "cli", gateway.OutgoingMessage{Text: "hello world"})
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", buf.String())
}

func TestCLIAdapter_Send_Markdown(t *testing.T) {
	var buf bytes.Buffer
	a := NewCLIAdapter(WithCLIWriter(&buf))

	err := a.Send(t.Context(), "cli", gateway.OutgoingMessage{
		Text:   "**bold** and *italic*",
		Format: "markdown",
	})
	require.NoError(t, err)
	assert.Equal(t, "bold and italic\n", buf.String())
}

func TestCLIAdapter_GetChatInfo(t *testing.T) {
	a := NewCLIAdapter()
	info, err := a.GetChatInfo(t.Context(), "cli")
	require.NoError(t, err)
	assert.Equal(t, "cli", info.ID)
	assert.Equal(t, "Terminal", info.Title)
	assert.Equal(t, "private", info.Type)
	assert.Equal(t, "cli", info.Platform)
}

func TestCLIAdapter_SupportedMedia(t *testing.T) {
	a := NewCLIAdapter()
	media := a.SupportedMedia()
	assert.Empty(t, media)
}

// collectMessages returns a buffered channel the read loop pushes to, along
// with a drain helper that waits for `want` messages (or times out).
func collectMessages(a *CLIAdapter, buf int) (chan gateway.IncomingMessage, func(int, time.Duration) []gateway.IncomingMessage) {
	ch := make(chan gateway.IncomingMessage, buf)
	a.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		ch <- msg
	})
	drain := func(want int, timeout time.Duration) []gateway.IncomingMessage {
		out := make([]gateway.IncomingMessage, 0, want)
		deadline := time.After(timeout)
		for len(out) < want {
			select {
			case m := <-ch:
				out = append(out, m)
			case <-deadline:
				return out
			}
		}
		return out
	}
	return ch, drain
}

func TestCLIAdapter_ReadLoop(t *testing.T) {
	input := "hello\nworld\n"
	r := strings.NewReader(input)
	a := NewCLIAdapter(WithCLIReader(r))

	_, drain := collectMessages(a, 4)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	require.NoError(t, a.Connect(ctx))

	received := drain(2, 2*time.Second)
	require.Len(t, received, 2)
	assert.Equal(t, "hello", received[0].Text)
	assert.Equal(t, "world", received[1].Text)
	assert.Equal(t, "cli", received[0].Platform)
	assert.Equal(t, "cli", received[0].ChatID)
}

func TestCLIAdapter_ReadLoop_SkipsEmptyLines(t *testing.T) {
	input := "\n  \nhello\n\n"
	r := strings.NewReader(input)
	a := NewCLIAdapter(WithCLIReader(r))

	_, drain := collectMessages(a, 4)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	require.NoError(t, a.Connect(ctx))

	received := drain(1, 2*time.Second)
	require.Len(t, received, 1)
	assert.Equal(t, "hello", received[0].Text)
}

func TestCLIAdapter_DisconnectTwice(t *testing.T) {
	a := NewCLIAdapter(WithCLIReader(strings.NewReader("")))

	err := a.Connect(t.Context())
	require.NoError(t, err)

	err = a.Disconnect(t.Context())
	require.NoError(t, err)

	// Second disconnect should not panic.
	err = a.Disconnect(t.Context())
	require.NoError(t, err)
}

func TestCLIAdapter_SendStripsMarkdown(t *testing.T) {
	var buf strings.Builder
	a := NewCLIAdapter(WithCLIWriter(&buf))
	err := a.Send(t.Context(), "", gateway.OutgoingMessage{
		Text: "# Title\n\n**Bold** and *italic* with `code` and [link](https://example.com)",
	})
	require.NoError(t, err)
	got := buf.String()
	assert.NotContains(t, got, "**")
	assert.NotContains(t, got, "# ")
	assert.NotContains(t, got, "`")
	assert.Contains(t, got, "Bold")
	assert.Contains(t, got, "italic")
	assert.Contains(t, got, "code")
	assert.Contains(t, got, "link")
	assert.NotContains(t, got, "example.com")
}
