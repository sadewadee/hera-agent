package platforms

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/sadewadee/hera/internal/gateway"
)

// SlackAdapter connects to Slack using Socket Mode for real-time events.
type SlackAdapter struct {
	BaseAdapter
	botToken string
	appToken string
	api      *slack.Client
	socket   *socketmode.Client
	logger   *slog.Logger
}

// NewSlackAdapter creates a Slack adapter. Both a bot token (xoxb-) and
// app-level token (xapp-) are required for Socket Mode.
func NewSlackAdapter(botToken, appToken string) *SlackAdapter {
	return &SlackAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "slack"},
		botToken:    botToken,
		appToken:    appToken,
		logger:      slog.Default(),
	}
}

// Connect initializes the Slack API client and starts Socket Mode.
func (s *SlackAdapter) Connect(ctx context.Context) error {
	api := slack.New(
		s.botToken,
		slack.OptionAppLevelToken(s.appToken),
	)

	socketClient := socketmode.New(api)
	s.api = api
	s.socket = socketClient
	s.SetConnected(true)

	go s.handleEvents(ctx)
	go func() {
		if err := socketClient.RunContext(ctx); err != nil {
			s.logger.Error("slack socket mode error", "error", err)
			s.SetConnected(false)
		}
	}()

	s.logger.Info("slack connected")
	return nil
}

// Disconnect closes the Socket Mode connection.
func (s *SlackAdapter) Disconnect(_ context.Context) error {
	s.SetConnected(false)
	// socketmode client stops when its context is cancelled.
	return nil
}

// Send delivers a message to a Slack channel.
func (s *SlackAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if s.api == nil {
		return fmt.Errorf("slack: not connected")
	}

	opts := []slack.MsgOption{
		slack.MsgOptionText(msg.Text, false),
	}

	if msg.ReplyTo != "" {
		opts = append(opts, slack.MsgOptionTS(msg.ReplyTo))
	}

	_, _, err := s.api.PostMessageContext(ctx, chatID, opts...)
	if err != nil {
		return fmt.Errorf("slack send: %w", err)
	}

	// Send file attachments.
	for _, m := range msg.Media {
		if m.Data != nil {
			params := slack.UploadFileParameters{
				Channel:  chatID,
				Filename: m.Caption,
				Reader:   bytesReader(m.Data),
				FileSize: len(m.Data),
			}
			if _, err := s.api.UploadFileContext(ctx, params); err != nil {
				s.logger.Warn("slack file upload failed", "error", err)
			}
		}
	}

	return nil
}

// GetChatInfo fetches channel information from Slack.
func (s *SlackAdapter) GetChatInfo(ctx context.Context, chatID string) (*gateway.ChatInfo, error) {
	if s.api == nil {
		return nil, fmt.Errorf("slack: not connected")
	}

	ch, err := s.api.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
		ChannelID: chatID,
	})
	if err != nil {
		return nil, fmt.Errorf("slack get channel: %w", err)
	}

	chatType := "group"
	if ch.IsIM {
		chatType = "private"
	} else if ch.IsChannel {
		chatType = "channel"
	}

	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    ch.Name,
		Type:     chatType,
		Platform: "slack",
	}, nil
}

// SupportedMedia returns the media types Slack supports.
func (s *SlackAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{
		gateway.MediaPhoto,
		gateway.MediaFile,
	}
}

// handleEvents processes incoming Socket Mode events.
func (s *SlackAdapter) handleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-s.socket.Events:
			if !ok {
				s.SetConnected(false)
				return
			}
			s.processEvent(ctx, evt)
		}
	}
}

// processEvent routes a Socket Mode event to the appropriate handler.
func (s *SlackAdapter) processEvent(ctx context.Context, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		s.socket.Ack(*evt.Request)
		s.handleEventsAPI(ctx, evt)
	default:
		// Acknowledge other event types to keep the connection alive.
		if evt.Request != nil {
			s.socket.Ack(*evt.Request)
		}
	}
}

// handleEventsAPI processes Slack Events API payloads (messages, etc.).
func (s *SlackAdapter) handleEventsAPI(ctx context.Context, evt socketmode.Event) {
	eventsAPI, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		return
	}

	switch ev := eventsAPI.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		// Ignore bot messages to avoid loops.
		if ev.BotID != "" {
			return
		}

		incoming := gateway.IncomingMessage{
			Platform:  "slack",
			ChatID:    ev.Channel,
			UserID:    ev.User,
			Username:  ev.User,
			Text:      ev.Text,
			Timestamp: time.Now(),
		}

		if ev.ThreadTimeStamp != "" {
			incoming.ReplyTo = ev.ThreadTimeStamp
		}

		// File attachments are available via the Message sub-object when present.
		if ev.Message != nil {
			for _, f := range ev.Message.Files {
				mediaType := gateway.MediaFile
				if len(f.Mimetype) > 6 && f.Mimetype[:6] == "image/" {
					mediaType = gateway.MediaPhoto
				}
				incoming.Media = append(incoming.Media, gateway.Media{
					Type:    mediaType,
					URL:     f.URLPrivateDownload,
					Caption: f.Name,
				})
			}
		}

		h := s.Handler()
		if h != nil {
			h(ctx, incoming)
		}
	}
}
