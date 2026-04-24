package platforms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestNewDiscordAdapter(t *testing.T) {
	a := NewDiscordAdapter("test-token")
	require.NotNil(t, a)
	assert.Equal(t, "discord", a.Name())
	assert.False(t, a.IsConnected())
}

func TestDiscordAdapter_SendNotConnected(t *testing.T) {
	a := NewDiscordAdapter("test-token")
	err := a.Send(t.Context(), "channel-123", gateway.OutgoingMessage{Text: "hello"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestDiscordAdapter_GetChatInfoNotConnected(t *testing.T) {
	a := NewDiscordAdapter("test-token")
	_, err := a.GetChatInfo(t.Context(), "channel-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestDiscordAdapter_SupportedMedia(t *testing.T) {
	a := NewDiscordAdapter("test-token")
	media := a.SupportedMedia()
	assert.Len(t, media, 2)
	assert.Contains(t, media, gateway.MediaPhoto)
	assert.Contains(t, media, gateway.MediaFile)
}

func TestDiscordAdapter_Disconnect(t *testing.T) {
	a := NewDiscordAdapter("test-token")
	a.SetConnected(true)

	err := a.Disconnect(t.Context())
	require.NoError(t, err)
	assert.False(t, a.IsConnected())
}

func TestIsImageContentType(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{"jpeg", "image/jpeg", true},
		{"png", "image/png", true},
		{"gif", "image/gif", true},
		{"webp", "image/webp", true},
		{"text", "text/plain", false},
		{"json", "application/json", false},
		{"empty", "", false},
		{"short", "img", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isImageContentType(tt.ct))
		})
	}
}

func TestBytesReader(t *testing.T) {
	data := []byte("hello world")
	r := bytesReader(data)

	buf := make([]byte, 5)
	n, err := r.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(buf))

	n, err = r.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, " worl", string(buf))

	buf = make([]byte, 10)
	n, err = r.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, "d", string(buf[:n]))

	// EOF
	_, err = r.Read(buf)
	assert.Error(t, err)
}
