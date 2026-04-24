package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestSendMessageTool_Name(t *testing.T) {
	tool := &SendMessageTool{}
	assert.Equal(t, "send_message", tool.Name())
}

func TestSendMessageTool_Description(t *testing.T) {
	tool := &SendMessageTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestSendMessageTool_Parameters(t *testing.T) {
	tool := &SendMessageTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestSendMessageTool_InvalidArgs(t *testing.T) {
	tool := &SendMessageTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSendMessageTool_NoGatewayFailsLoudly(t *testing.T) {
	// This is the v0.14.1 hallucination regression test: calling
	// send_message without a gateway used to return a fake success
	// ("Message queued for telegram:chat123 (6 chars)"). The real
	// implementation must refuse with an error so the LLM stops
	// reporting fake uploads to the user.
	tool := &SendMessageTool{}
	args, _ := json.Marshal(sendMessageArgs{Platform: "telegram", ChatID: "chat123", Text: "Hello!"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "nil gateway must surface as tool error, not silent success")
	assert.Contains(t, result.Content, "no gateway available")
}

func TestSendMessageTool_RequiresPlatformAndChatID(t *testing.T) {
	tool := &SendMessageTool{}
	cases := []struct {
		name    string
		args    sendMessageArgs
		wantMsg string
	}{
		{"missing platform", sendMessageArgs{ChatID: "c", Text: "hi"}, "platform is required"},
		{"missing chat_id", sendMessageArgs{Platform: "p", Text: "hi"}, "chat_id is required"},
		{"no text and no attachments", sendMessageArgs{Platform: "p", ChatID: "c"}, "either text or attachments"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(tc.args)
			result, err := tool.Execute(context.Background(), raw)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, result.Content, tc.wantMsg)
		})
	}
}

// fakeAdapter captures whatever gets dispatched to Send so the test can
// assert the tool built the right OutgoingMessage (text + media list).
type fakeAdapter struct {
	mu        sync.Mutex
	name      string
	connected bool
	sent      []struct {
		chatID string
		msg    gateway.OutgoingMessage
	}
	sendErr error
}

func (f *fakeAdapter) Name() string                     { return f.name }
func (f *fakeAdapter) Connect(context.Context) error    { f.connected = true; return nil }
func (f *fakeAdapter) Disconnect(context.Context) error { f.connected = false; return nil }
func (f *fakeAdapter) IsConnected() bool                { return f.connected }
func (f *fakeAdapter) OnMessage(gateway.MessageHandler) {}
func (f *fakeAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaFile, gateway.MediaPhoto}
}
func (f *fakeAdapter) GetChatInfo(context.Context, string) (*gateway.ChatInfo, error) {
	return nil, nil
}
func (f *fakeAdapter) Send(_ context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, struct {
		chatID string
		msg    gateway.OutgoingMessage
	}{chatID, msg})
	return nil
}

func TestSendMessageTool_DispatchesToConnectedAdapter(t *testing.T) {
	gw := gateway.NewGateway(gateway.GatewayOptions{})
	adapter := &fakeAdapter{name: "telegram", connected: true}
	gw.AddAdapter(adapter)

	tool := &SendMessageTool{gw: gw}
	args, _ := json.Marshal(sendMessageArgs{
		Platform: "telegram",
		ChatID:   "chat123",
		Text:     "Hello!",
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "got error result: %s", result.Content)
	assert.Contains(t, result.Content, "Sent to telegram:chat123")

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	require.Len(t, adapter.sent, 1)
	assert.Equal(t, "chat123", adapter.sent[0].chatID)
	assert.Equal(t, "Hello!", adapter.sent[0].msg.Text)
}

func TestSendMessageTool_UploadsLocalAttachmentAndDetectsType(t *testing.T) {
	// Seed a .md and a .png on disk so the tool has something real to
	// attach. Verify the tool passes absolute paths through and picks
	// the right MediaType per extension.
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "article.md")
	pngPath := filepath.Join(dir, "cover.png")
	require.NoError(t, os.WriteFile(mdPath, []byte("# hello"), 0o644))
	require.NoError(t, os.WriteFile(pngPath, []byte("\x89PNG\r\n\x1a\n"), 0o644))

	gw := gateway.NewGateway(gateway.GatewayOptions{})
	adapter := &fakeAdapter{name: "telegram", connected: true}
	gw.AddAdapter(adapter)

	tool := &SendMessageTool{gw: gw}
	args, _ := json.Marshal(sendMessageArgs{
		Platform:    "telegram",
		ChatID:      "chat123",
		Text:        "batch of articles",
		Attachments: []string{mdPath, pngPath},
		Caption:     "v0.14.3 regression",
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "got error result: %s", result.Content)
	assert.Contains(t, result.Content, "attachments=2")

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	require.Len(t, adapter.sent, 1)
	media := adapter.sent[0].msg.Media
	require.Len(t, media, 2)
	assert.Equal(t, gateway.MediaFile, media[0].Type, ".md should detect as MediaFile")
	assert.Equal(t, mdPath, media[0].URL, "tool must normalize to absolute path")
	assert.Equal(t, "v0.14.3 regression", media[0].Caption)
	assert.Equal(t, gateway.MediaPhoto, media[1].Type, ".png should detect as MediaPhoto")
	assert.Equal(t, pngPath, media[1].URL)
}

func TestSendMessageTool_RefusesMissingAttachment(t *testing.T) {
	gw := gateway.NewGateway(gateway.GatewayOptions{})
	adapter := &fakeAdapter{name: "telegram", connected: true}
	gw.AddAdapter(adapter)

	tool := &SendMessageTool{gw: gw}
	args, _ := json.Marshal(sendMessageArgs{
		Platform:    "telegram",
		ChatID:      "chat123",
		Text:        "oops",
		Attachments: []string{"/definitely/does/not/exist.md"},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "does/not/exist.md")
	// Must NOT have called Send.
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	assert.Empty(t, adapter.sent)
}

func TestSendMessageTool_RefusesDisconnectedAdapter(t *testing.T) {
	gw := gateway.NewGateway(gateway.GatewayOptions{})
	adapter := &fakeAdapter{name: "telegram", connected: false}
	gw.AddAdapter(adapter)

	tool := &SendMessageTool{gw: gw}
	args, _ := json.Marshal(sendMessageArgs{Platform: "telegram", ChatID: "c", Text: "x"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "not connected")
}

func TestSendMessageTool_SurfacesAdapterError(t *testing.T) {
	gw := gateway.NewGateway(gateway.GatewayOptions{})
	adapter := &fakeAdapter{name: "telegram", connected: true, sendErr: errors.New("telegram 429")}
	gw.AddAdapter(adapter)

	tool := &SendMessageTool{gw: gw}
	args, _ := json.Marshal(sendMessageArgs{Platform: "telegram", ChatID: "c", Text: "x"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "telegram 429")
}

func TestDetectMediaType(t *testing.T) {
	cases := map[string]gateway.MediaType{
		"/tmp/foo.jpg":  gateway.MediaPhoto,
		"/tmp/foo.PNG":  gateway.MediaPhoto,
		"/tmp/foo.webp": gateway.MediaPhoto,
		"/tmp/foo.mp3":  gateway.MediaAudio,
		"/tmp/foo.ogg":  gateway.MediaVoice,
		"/tmp/foo.opus": gateway.MediaVoice,
		"/tmp/foo.mp4":  gateway.MediaVideo,
		"/tmp/foo.webm": gateway.MediaVideo,
		"/tmp/foo.md":   gateway.MediaFile,
		"/tmp/foo.pdf":  gateway.MediaFile,
		"/tmp/noext":    gateway.MediaFile,
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			got := detectMediaType(path)
			assert.Equal(t, want, got)
		})
	}
}
