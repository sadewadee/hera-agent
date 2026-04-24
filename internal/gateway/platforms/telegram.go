package platforms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sadewadee/hera/internal/gateway"
)

// TelegramOptions configures optional Telegram adapter behaviour. Zero
// value selects long-polling with no webhook, which matches the default
// adapter created by NewTelegramAdapter.
type TelegramOptions struct {
	// Mode is "polling" (default) or "webhook".
	Mode string
	// WebhookURL is the publicly reachable HTTPS URL Telegram will call.
	// Required when Mode == "webhook".
	WebhookURL string
	// WebhookAddr is the local listen address (e.g. ":8443") the adapter
	// serves. Required when Mode == "webhook".
	WebhookAddr string
	// WebhookPath is the URL path under WebhookURL; defaults to "/" +
	// the bot token so two bots can share one host.
	WebhookPath string
}

// TelegramAdapter connects to the Telegram Bot API using either long
// polling (default) or webhook mode.
type TelegramAdapter struct {
	BaseAdapter
	token   string
	bot     *tgbotapi.BotAPI
	logger  *slog.Logger
	opts    TelegramOptions
	httpSrv *http.Server // only set in webhook mode
}

// NewTelegramAdapter creates a long-polling Telegram adapter.
func NewTelegramAdapter(token string) *TelegramAdapter {
	return &TelegramAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "telegram"},
		token:       token,
		logger:      slog.Default(),
	}
}

// NewTelegramAdapterWithOptions creates a Telegram adapter with explicit
// mode/webhook configuration. Falls back to polling if webhook setup
// fails (e.g. URL unreachable) so the bot stays online.
func NewTelegramAdapterWithOptions(token string, opts TelegramOptions) *TelegramAdapter {
	a := NewTelegramAdapter(token)
	a.opts = opts
	return a
}

// Connect authenticates the bot and starts receiving updates in the
// configured mode (polling or webhook).
func (t *TelegramAdapter) Connect(ctx context.Context) error {
	bot, err := tgbotapi.NewBotAPI(t.token)
	if err != nil {
		return fmt.Errorf("telegram connect: %w", err)
	}
	t.bot = bot
	t.SetConnected(true)
	t.logger.Info("telegram connected", "bot", bot.Self.UserName, "mode", t.mode())

	if t.mode() == "webhook" {
		if err := t.startWebhook(ctx); err != nil {
			t.logger.Warn("telegram: webhook setup failed, falling back to polling", "error", err)
			go t.pollUpdates(ctx)
		}
		return nil
	}
	go t.pollUpdates(ctx)
	return nil
}

// mode returns the effective mode, defaulting to polling.
func (t *TelegramAdapter) mode() string {
	if strings.EqualFold(t.opts.Mode, "webhook") {
		return "webhook"
	}
	return "polling"
}

// startWebhook registers the webhook with Telegram, starts a local HTTP
// server, and dispatches incoming updates. Returns an error when
// registration or the local server can't start — caller falls back to
// polling.
func (t *TelegramAdapter) startWebhook(ctx context.Context) error {
	if t.opts.WebhookURL == "" {
		return fmt.Errorf("webhook mode requires webhook_url")
	}
	if t.opts.WebhookAddr == "" {
		return fmt.Errorf("webhook mode requires webhook_addr")
	}

	path := t.opts.WebhookPath
	if path == "" {
		path = "/" + t.token
	}
	fullURL := strings.TrimRight(t.opts.WebhookURL, "/") + path

	wh, err := tgbotapi.NewWebhook(fullURL)
	if err != nil {
		return fmt.Errorf("build webhook: %w", err)
	}
	if _, err := t.bot.Request(wh); err != nil {
		return fmt.Errorf("register webhook: %w", err)
	}

	updates := t.bot.ListenForWebhook(path)
	mux := http.NewServeMux()
	mux.Handle(path, http.DefaultServeMux)

	t.httpSrv = &http.Server{
		Addr:              t.opts.WebhookAddr,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := t.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.logger.Warn("telegram webhook server exited", "error", err)
			t.SetConnected(false)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-updates:
				if !ok {
					t.SetConnected(false)
					return
				}
				if update.CallbackQuery != nil {
					t.handleCallback(ctx, update.CallbackQuery)
					continue
				}
				if update.Message == nil {
					continue
				}
				t.handleUpdate(ctx, update)
			}
		}
	}()

	return nil
}

// Disconnect stops polling/webhook, unregisters the webhook if set, and
// shuts down the HTTP server. Safe to call repeatedly.
func (t *TelegramAdapter) Disconnect(_ context.Context) error {
	if t.bot != nil {
		t.bot.StopReceivingUpdates()
		if t.mode() == "webhook" {
			// Best-effort: tell Telegram to stop pushing.
			if _, err := t.bot.Request(tgbotapi.DeleteWebhookConfig{}); err != nil {
				t.logger.Warn("telegram: delete webhook failed", "error", err)
			}
		}
	}
	if t.httpSrv != nil {
		_ = t.httpSrv.Close()
		t.httpSrv = nil
	}
	t.SetConnected(false)
	return nil
}

// Send delivers an outgoing message to a Telegram chat.
func (t *TelegramAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if t.bot == nil {
		return fmt.Errorf("telegram: not connected")
	}

	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: invalid chat ID %q: %w", chatID, err)
	}

	// Send media attachments first. Errors from individual uploads are
	// accumulated and returned so callers (e.g. the send_message tool)
	// can surface real failure to the LLM instead of silently reporting
	// success. Previously errors were only Warn-logged — that let the
	// LLM hallucinate "uploaded" after a 429/413/permission failure.
	var mediaErrs []error
	for _, m := range msg.Media {
		if sendErr := t.sendMedia(id, m); sendErr != nil {
			t.logger.Warn("telegram send media failed", "error", sendErr)
			mediaErrs = append(mediaErrs, sendErr)
		}
	}
	if len(mediaErrs) > 0 {
		return fmt.Errorf("telegram: %d/%d attachment(s) failed: %w",
			len(mediaErrs), len(msg.Media), errors.Join(mediaErrs...))
	}

	// Send text message. Format=="markdown" (LLM output) is converted to
	// Telegram HTML because MarkdownV2 requires strict escaping of 18
	// reserved chars that LLMs don't produce. Format=="html" is passed
	// through unchanged (caller is responsible for escaping).
	if msg.Text != "" {
		text := msg.Text
		var parseMode string
		switch msg.Format {
		case "markdown":
			text = markdownToTelegramHTML(text)
			parseMode = tgbotapi.ModeHTML
		case "html":
			parseMode = tgbotapi.ModeHTML
		}
		tgMsg := tgbotapi.NewMessage(id, text)
		tgMsg.ParseMode = parseMode
		if msg.ReplyTo != "" {
			replyID, parseErr := strconv.Atoi(msg.ReplyTo)
			if parseErr == nil {
				tgMsg.ReplyToMessageID = replyID
			}
		}
		if len(msg.Buttons) > 0 {
			tgMsg.ReplyMarkup = buildTelegramKeyboard(msg.Buttons)
		}
		if _, err := t.bot.Send(tgMsg); err != nil {
			return fmt.Errorf("telegram send text: %w", err)
		}
	}

	return nil
}

// buildTelegramKeyboard renders a [][]gateway.Button as an inline
// keyboard Telegram can display under the message.
func buildTelegramKeyboard(rows [][]gateway.Button) tgbotapi.InlineKeyboardMarkup {
	kbRows := make([][]tgbotapi.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		kbRow := make([]tgbotapi.InlineKeyboardButton, 0, len(row))
		for _, b := range row {
			kbRow = append(kbRow, tgbotapi.NewInlineKeyboardButtonData(b.Text, b.Data))
		}
		kbRows = append(kbRows, kbRow)
	}
	return tgbotapi.NewInlineKeyboardMarkup(kbRows...)
}

// GetChatInfo fetches chat metadata from Telegram.
func (t *TelegramAdapter) GetChatInfo(ctx context.Context, chatID string) (*gateway.ChatInfo, error) {
	if t.bot == nil {
		return nil, fmt.Errorf("telegram: not connected")
	}

	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: invalid chat ID %q: %w", chatID, err)
	}

	chatCfg := tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: id}}
	chat, err := t.bot.GetChat(chatCfg)
	if err != nil {
		return nil, fmt.Errorf("telegram get chat: %w", err)
	}

	chatType := "private"
	if chat.Type == "group" || chat.Type == "supergroup" {
		chatType = "group"
	} else if chat.Type == "channel" {
		chatType = "channel"
	}

	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    chat.Title,
		Type:     chatType,
		Platform: "telegram",
	}, nil
}

// SupportedMedia returns the media types Telegram supports.
func (t *TelegramAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{
		gateway.MediaPhoto,
		gateway.MediaAudio,
		gateway.MediaVideo,
		gateway.MediaFile,
		gateway.MediaVoice,
	}
}

// pollUpdates long-polls Telegram for new messages and dispatches them.
func (t *TelegramAdapter) pollUpdates(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := t.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				t.SetConnected(false)
				return
			}
			if update.CallbackQuery != nil {
				t.handleCallback(ctx, update.CallbackQuery)
				continue
			}
			if update.Message == nil {
				continue
			}
			t.handleUpdate(ctx, update)
		}
	}
}

// handleCallback converts an inline-button click into an IncomingMessage
// whose Text is the button's Data payload. This lets the same agent
// handler that processes normal messages react to button presses.
// Also acknowledges the callback to Telegram so the spinner stops.
func (t *TelegramAdapter) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if cb == nil {
		return
	}
	// Ack so Telegram dismisses the loading indicator.
	if _, err := t.bot.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
		t.logger.Warn("telegram: callback ack failed", "error", err)
	}

	var chatID int64
	if cb.Message != nil {
		chatID = cb.Message.Chat.ID
	}
	incoming := gateway.IncomingMessage{
		Platform:  "telegram",
		ChatID:    strconv.FormatInt(chatID, 10),
		UserID:    strconv.FormatInt(cb.From.ID, 10),
		Username:  cb.From.UserName,
		Text:      cb.Data,
		Timestamp: time.Now(),
	}
	if cb.Message != nil {
		incoming.ReplyTo = strconv.Itoa(cb.Message.MessageID)
	}

	if h := t.Handler(); h != nil {
		h(ctx, incoming)
	}
}

// handleUpdate converts a Telegram update into an IncomingMessage.
func (t *TelegramAdapter) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	msg := update.Message
	incoming := gateway.IncomingMessage{
		Platform:  "telegram",
		ChatID:    strconv.FormatInt(msg.Chat.ID, 10),
		UserID:    strconv.FormatInt(msg.From.ID, 10),
		Username:  msg.From.UserName,
		Text:      msg.Text,
		Timestamp: time.Unix(int64(msg.Date), 0),
	}

	if msg.Caption != "" && incoming.Text == "" {
		incoming.Text = msg.Caption
	}

	if msg.ReplyToMessage != nil {
		incoming.ReplyTo = strconv.Itoa(msg.ReplyToMessage.MessageID)
	}

	// Collect media.
	incoming.Media = t.extractMedia(msg)

	h := t.Handler()
	if h != nil {
		h(ctx, incoming)
	}
}

// extractMedia pulls media attachments from a Telegram message.
func (t *TelegramAdapter) extractMedia(msg *tgbotapi.Message) []gateway.Media {
	var media []gateway.Media

	if msg.Photo != nil && len(msg.Photo) > 0 {
		// Use the highest resolution photo.
		best := msg.Photo[len(msg.Photo)-1]
		media = append(media, gateway.Media{
			Type:    gateway.MediaPhoto,
			URL:     best.FileID,
			Caption: msg.Caption,
		})
	}
	if msg.Audio != nil {
		media = append(media, gateway.Media{
			Type:    gateway.MediaAudio,
			URL:     msg.Audio.FileID,
			Caption: msg.Caption,
		})
	}
	if msg.Video != nil {
		media = append(media, gateway.Media{
			Type:    gateway.MediaVideo,
			URL:     msg.Video.FileID,
			Caption: msg.Caption,
		})
	}
	if msg.Document != nil {
		media = append(media, gateway.Media{
			Type:    gateway.MediaFile,
			URL:     msg.Document.FileID,
			Caption: msg.Caption,
		})
	}
	if msg.Voice != nil {
		media = append(media, gateway.Media{
			Type:    gateway.MediaVoice,
			URL:     msg.Voice.FileID,
			Caption: msg.Caption,
		})
	}

	return media
}

// sendMedia sends a single media attachment to a Telegram chat. The
// attachment source is picked by telegramMediaFile: raw bytes > local
// path > remote URL, in that priority.
func (t *TelegramAdapter) sendMedia(chatID int64, m gateway.Media) error {
	src, err := telegramMediaFile(m)
	if err != nil {
		return err
	}
	switch m.Type {
	case gateway.MediaPhoto:
		msg := tgbotapi.NewPhoto(chatID, src)
		msg.Caption = m.Caption
		_, err := t.bot.Send(msg)
		return err
	case gateway.MediaAudio:
		msg := tgbotapi.NewAudio(chatID, src)
		msg.Caption = m.Caption
		_, err := t.bot.Send(msg)
		return err
	case gateway.MediaVideo:
		msg := tgbotapi.NewVideo(chatID, src)
		msg.Caption = m.Caption
		_, err := t.bot.Send(msg)
		return err
	case gateway.MediaFile:
		msg := tgbotapi.NewDocument(chatID, src)
		msg.Caption = m.Caption
		_, err := t.bot.Send(msg)
		return err
	case gateway.MediaVoice:
		msg := tgbotapi.NewVoice(chatID, src)
		msg.Caption = m.Caption
		_, err := t.bot.Send(msg)
		return err
	default:
		return fmt.Errorf("unsupported media type: %s", m.Type)
	}
}

// telegramMediaFile returns the tgbotapi RequestFileData for a
// gateway.Media, picking the first available source: raw Data bytes
// (in-memory upload), a local filesystem path (multipart upload), or a
// remote HTTP/HTTPS URL (Telegram fetches server-side).
func telegramMediaFile(m gateway.Media) (tgbotapi.RequestFileData, error) {
	if len(m.Data) > 0 {
		return tgbotapi.FileBytes{Name: "attachment", Bytes: m.Data}, nil
	}
	if m.URL == "" {
		return nil, fmt.Errorf("media has no Data, URL, or path")
	}
	if strings.HasPrefix(m.URL, "http://") || strings.HasPrefix(m.URL, "https://") {
		return tgbotapi.FileURL(m.URL), nil
	}
	path := strings.TrimPrefix(m.URL, "file://")
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("local attachment path must be absolute, got %q", m.URL)
	}
	return tgbotapi.FilePath(path), nil
}

// markdownToTelegramHTML converts standard markdown (as produced by LLMs) to
// Telegram HTML. It protects fenced and inline code from further substitution,
// then HTML-escapes the remaining prose before applying bold, italic, link,
// and heading transforms. Only tags Telegram accepts are emitted.
var (
	reTGCodeBlock  = regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]*)\\n?(.*?)```")
	reTGInlineCode = regexp.MustCompile("`([^`\\n]+)`")
	reTGBoldStar   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reTGBoldUnder  = regexp.MustCompile(`__(.+?)__`)
	reTGItalicStar = regexp.MustCompile(`\*([^*\n]+?)\*`)
	reTGItalicUnd  = regexp.MustCompile(`(^|[^\w])_([^_\n]+?)_([^\w]|$)`)
	reTGLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reTGHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`)
)

func markdownToTelegramHTML(text string) string {
	htmlEscape := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		return s
	}

	// 1. Stash fenced code blocks so later passes don't touch their contents.
	type stash struct{ lang, body string }
	var codeBlocks []stash
	text = reTGCodeBlock.ReplaceAllStringFunc(text, func(match string) string {
		parts := reTGCodeBlock.FindStringSubmatch(match)
		codeBlocks = append(codeBlocks, stash{lang: parts[1], body: parts[2]})
		return fmt.Sprintf("\x00CB%d\x00", len(codeBlocks)-1)
	})

	// 2. Stash inline code the same way.
	var inlineCodes []string
	text = reTGInlineCode.ReplaceAllStringFunc(text, func(match string) string {
		parts := reTGInlineCode.FindStringSubmatch(match)
		inlineCodes = append(inlineCodes, parts[1])
		return fmt.Sprintf("\x00IC%d\x00", len(inlineCodes)-1)
	})

	// 3. Escape any raw HTML metacharacters in the remaining prose.
	text = htmlEscape(text)

	// 4. Apply prose-level markdown transforms. Bold must precede italic so
	//    "**x**" doesn't get eaten by the single-star italic pattern.
	text = reTGBoldStar.ReplaceAllString(text, "<b>$1</b>")
	text = reTGBoldUnder.ReplaceAllString(text, "<b>$1</b>")
	text = reTGItalicStar.ReplaceAllString(text, "<i>$1</i>")
	text = reTGItalicUnd.ReplaceAllString(text, "$1<i>$2</i>$3")
	text = reTGLink.ReplaceAllString(text, `<a href="$2">$1</a>`)
	text = reTGHeading.ReplaceAllString(text, "<b>$1</b>")

	// 5. Restore inline code with escaped content.
	for i, body := range inlineCodes {
		text = strings.Replace(text,
			fmt.Sprintf("\x00IC%d\x00", i),
			"<code>"+htmlEscape(body)+"</code>", 1)
	}

	// 6. Restore fenced code blocks with escaped content.
	for i, cb := range codeBlocks {
		body := htmlEscape(cb.body)
		var replacement string
		if cb.lang != "" {
			replacement = fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, cb.lang, body)
		} else {
			replacement = "<pre>" + body + "</pre>"
		}
		text = strings.Replace(text, fmt.Sprintf("\x00CB%d\x00", i), replacement, 1)
	}

	return text
}
