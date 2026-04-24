package platforms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestNewSlackAdapter(t *testing.T) {
	a := NewSlackAdapter("xoxb-test", "xapp-test")
	require.NotNil(t, a)
	assert.Equal(t, "slack", a.Name())
	assert.False(t, a.IsConnected())
}

func TestSlackAdapter_SendNotConnected(t *testing.T) {
	a := NewSlackAdapter("xoxb-test", "xapp-test")
	err := a.Send(t.Context(), "C123", gateway.OutgoingMessage{Text: "hello"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestSlackAdapter_GetChatInfoNotConnected(t *testing.T) {
	a := NewSlackAdapter("xoxb-test", "xapp-test")
	_, err := a.GetChatInfo(t.Context(), "C123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestSlackAdapter_SupportedMedia(t *testing.T) {
	a := NewSlackAdapter("xoxb-test", "xapp-test")
	media := a.SupportedMedia()
	assert.Len(t, media, 2)
	assert.Contains(t, media, gateway.MediaPhoto)
	assert.Contains(t, media, gateway.MediaFile)
}

func TestSlackAdapter_Disconnect(t *testing.T) {
	a := NewSlackAdapter("xoxb-test", "xapp-test")
	a.SetConnected(true)

	err := a.Disconnect(t.Context())
	require.NoError(t, err)
	assert.False(t, a.IsConnected())
}
