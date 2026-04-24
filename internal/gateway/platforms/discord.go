package platforms

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/sadewadee/hera/internal/gateway"
)

// DiscordAdapter connects to Discord via WebSocket using discordgo.
type DiscordAdapter struct {
	BaseAdapter
	token   string
	session *discordgo.Session
	logger  *slog.Logger
}

// NewDiscordAdapter creates a Discord adapter with the given bot token.
func NewDiscordAdapter(token string) *DiscordAdapter {
	return &DiscordAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "discord"},
		token:       token,
		logger:      slog.Default(),
	}
}

// Connect opens the Discord WebSocket gateway and registers handlers.
func (d *DiscordAdapter) Connect(ctx context.Context) error {
	sess, err := discordgo.New("Bot " + d.token)
	if err != nil {
		return fmt.Errorf("discord create session: %w", err)
	}

	sess.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	sess.AddHandler(d.onMessageCreate)
	sess.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		d.logger.Info("discord connected", "user", s.State.User.Username)
	})
	sess.AddHandler(func(s *discordgo.Session, dc *discordgo.Disconnect) {
		d.SetConnected(false)
		d.logger.Warn("discord disconnected")
	})

	if err := sess.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}

	d.session = sess
	d.SetConnected(true)
	return nil
}

// Disconnect closes the Discord session.
func (d *DiscordAdapter) Disconnect(_ context.Context) error {
	d.SetConnected(false)
	if d.session != nil {
		return d.session.Close()
	}
	return nil
}

// Send delivers a message to a Discord channel.
func (d *DiscordAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if d.session == nil {
		return fmt.Errorf("discord: not connected")
	}

	// Send file attachments.
	if len(msg.Media) > 0 {
		for _, m := range msg.Media {
			if m.Data != nil {
				file := &discordgo.File{
					Name:   m.Caption,
					Reader: bytesReader(m.Data),
				}
				_, err := d.session.ChannelMessageSendComplex(chatID, &discordgo.MessageSend{
					Files: []*discordgo.File{file},
				})
				if err != nil {
					d.logger.Warn("discord send file failed", "error", err)
				}
			}
		}
	}

	// Send text.
	if msg.Text != "" {
		text := msg.Text
		if msg.Format == "markdown" {
			// Discord natively supports markdown, pass through.
			text = msg.Text
		}

		var ref *discordgo.MessageReference
		if msg.ReplyTo != "" {
			ref = &discordgo.MessageReference{MessageID: msg.ReplyTo}
		}

		_, err := d.session.ChannelMessageSendComplex(chatID, &discordgo.MessageSend{
			Content:   text,
			Reference: ref,
		})
		if err != nil {
			return fmt.Errorf("discord send text: %w", err)
		}
	}

	return nil
}

// GetChatInfo fetches channel information from Discord.
func (d *DiscordAdapter) GetChatInfo(ctx context.Context, chatID string) (*gateway.ChatInfo, error) {
	if d.session == nil {
		return nil, fmt.Errorf("discord: not connected")
	}

	ch, err := d.session.Channel(chatID)
	if err != nil {
		return nil, fmt.Errorf("discord get channel: %w", err)
	}

	chatType := "group"
	if ch.Type == discordgo.ChannelTypeDM {
		chatType = "private"
	}

	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    ch.Name,
		Type:     chatType,
		Platform: "discord",
	}, nil
}

// SupportedMedia returns the media types Discord supports.
func (d *DiscordAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{
		gateway.MediaPhoto,
		gateway.MediaFile,
	}
}

// onMessageCreate handles Discord MESSAGE_CREATE events.
func (d *DiscordAdapter) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself.
	if m.Author.ID == s.State.User.ID {
		return
	}

	incoming := gateway.IncomingMessage{
		Platform:  "discord",
		ChatID:    m.ChannelID,
		UserID:    m.Author.ID,
		Username:  m.Author.Username,
		Text:      m.Content,
		Timestamp: time.Now(),
	}

	if m.MessageReference != nil {
		incoming.ReplyTo = m.MessageReference.MessageID
	}

	// Collect attachments.
	for _, att := range m.Attachments {
		mediaType := gateway.MediaFile
		if isImageContentType(att.ContentType) {
			mediaType = gateway.MediaPhoto
		}
		incoming.Media = append(incoming.Media, gateway.Media{
			Type:    mediaType,
			URL:     att.URL,
			Caption: att.Filename,
		})
	}

	h := d.Handler()
	if h != nil {
		h(context.Background(), incoming)
	}
}

// isImageContentType checks if a MIME type indicates an image.
func isImageContentType(ct string) bool {
	return len(ct) > 6 && ct[:6] == "image/"
}

// bytesReader wraps a byte slice in a minimal io.Reader.
type bytesReaderImpl struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: data}
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
