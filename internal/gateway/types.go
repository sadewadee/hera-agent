package gateway

import (
	"context"
	"time"
)

// MediaType represents supported media types.
type MediaType string

const (
	MediaPhoto MediaType = "photo"
	MediaAudio MediaType = "audio"
	MediaVideo MediaType = "video"
	MediaFile  MediaType = "file"
	MediaVoice MediaType = "voice"
)

// Media represents a media attachment.
type Media struct {
	Type    MediaType `json:"type"`
	URL     string    `json:"url,omitempty"`
	Data    []byte    `json:"data,omitempty"`
	Caption string    `json:"caption,omitempty"`
}

// OutgoingMessage is a message sent from the agent to a platform.
type OutgoingMessage struct {
	Text    string     `json:"text"`
	Media   []Media    `json:"media,omitempty"`
	ReplyTo string     `json:"reply_to,omitempty"`
	Format  string     `json:"format,omitempty"`  // "markdown", "plain", "html"
	Buttons [][]Button `json:"buttons,omitempty"` // optional inline buttons; rendered per-adapter
}

// Button is a single interactive button. Platforms that support inline
// buttons (Telegram, Slack) render them natively. Platforms without
// button support (CLI) should fall back to showing Text plus a hint
// to type the Data value back.
type Button struct {
	Text string `json:"text"` // label shown to the user
	Data string `json:"data"` // callback payload delivered back as an IncomingMessage
}

// IncomingMessage is a message received from a platform.
type IncomingMessage struct {
	Platform  string    `json:"platform"`
	ChatID    string    `json:"chat_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Media     []Media   `json:"media,omitempty"`
	ReplyTo   string    `json:"reply_to,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatInfo describes a chat/channel on a platform.
type ChatInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"` // "private", "group", "channel"
	Platform string `json:"platform"`
}

// MessageHandler processes an incoming message.
type MessageHandler func(ctx context.Context, msg IncomingMessage)

// PlatformAdapter is the interface all platform adapters must implement.
type PlatformAdapter interface {
	Name() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Send(ctx context.Context, chatID string, msg OutgoingMessage) error
	OnMessage(handler MessageHandler)
	GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error)
	SupportedMedia() []MediaType
}
