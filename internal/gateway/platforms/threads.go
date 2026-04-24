package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

const (
	threadsBaseURL    = "https://graph.threads.net"
	threadsAPIVersion = "v1.0"
)

// ThreadsAdapter posts and fetches content via the Meta Threads Graph API.
// It is outbound-capable; inbound events arrive via the webhook adapter.
type ThreadsAdapter struct {
	BaseAdapter
	accessToken string
	httpClient  *http.Client
	logger      *slog.Logger
}

// NewThreadsAdapter creates a Threads adapter using the given access token.
func NewThreadsAdapter(accessToken string) *ThreadsAdapter {
	return &ThreadsAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "threads"},
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		logger:      slog.Default(),
	}
}

// Connect validates the access token by requesting the authenticated user profile.
func (a *ThreadsAdapter) Connect(ctx context.Context) error {
	if a.accessToken == "" {
		return fmt.Errorf("threads: access token required")
	}

	endpoint := fmt.Sprintf("%s/%s/me?fields=id,username", threadsBaseURL, threadsAPIVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("threads connect request: %w", err)
	}
	a.setAuth(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("threads connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("threads connect: %s: %s", resp.Status, string(body))
	}

	var me struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return fmt.Errorf("threads decode me: %w", err)
	}

	a.SetConnected(true)
	a.logger.Info("threads connected", "username", me.Username)
	return nil
}

// Disconnect marks the adapter as offline. No session is held, so this is a no-op.
func (a *ThreadsAdapter) Disconnect(_ context.Context) error {
	a.SetConnected(false)
	return nil
}

// Send publishes a post. If chatID is non-empty, it is treated as a thread ID
// and the message is sent as a reply; otherwise a new top-level post is created.
func (a *ThreadsAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !a.IsConnected() {
		return fmt.Errorf("threads: not connected")
	}
	if msg.Text == "" {
		return fmt.Errorf("threads: empty text")
	}

	mediaType := "TEXT"
	var mediaURL string
	if len(msg.Media) > 0 {
		m := msg.Media[0]
		switch m.Type {
		case gateway.MediaPhoto:
			mediaType = "IMAGE"
		case gateway.MediaVideo:
			mediaType = "VIDEO"
		}
		mediaURL = m.URL
	}

	containerID, err := a.createContainer(ctx, chatID, msg.Text, mediaType, mediaURL, msg.ReplyTo)
	if err != nil {
		return err
	}
	return a.publishContainer(ctx, containerID)
}

// GetChatInfo fetches metadata for a thread by ID.
func (a *ThreadsAdapter) GetChatInfo(ctx context.Context, chatID string) (*gateway.ChatInfo, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("threads: not connected")
	}

	endpoint := fmt.Sprintf("%s/%s/%s?fields=id,text,timestamp", threadsBaseURL, threadsAPIVersion, chatID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("threads get thread request: %w", err)
	}
	a.setAuth(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("threads get thread: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("threads get thread: %s: %s", resp.Status, string(body))
	}

	var thread struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&thread); err != nil {
		return nil, fmt.Errorf("threads decode thread: %w", err)
	}

	title := thread.Text
	if len(title) > 60 {
		title = title[:60] + "..."
	}
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    title,
		Type:     "channel",
		Platform: "threads",
	}, nil
}

// SupportedMedia returns the media types Threads accepts for outbound posts.
func (a *ThreadsAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaVideo}
}

// createContainer builds an unpublished post container and returns its ID.
// If replyToThreadID is set, the container is created as a reply.
func (a *ThreadsAdapter) createContainer(ctx context.Context, parentThreadID, text, mediaType, mediaURL, replyToThreadID string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/me/threads", threadsBaseURL, threadsAPIVersion)

	values := url.Values{}
	values.Set("media_type", mediaType)
	values.Set("text", text)
	if mediaURL != "" {
		switch mediaType {
		case "IMAGE":
			values.Set("image_url", mediaURL)
		case "VIDEO":
			values.Set("video_url", mediaURL)
		}
	}
	// Either parentThreadID (conversation chatID) or explicit ReplyTo marks this a reply.
	if replyToThreadID != "" {
		values.Set("reply_to_id", replyToThreadID)
	} else if parentThreadID != "" {
		values.Set("reply_to_id", parentThreadID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("threads create container request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	a.setAuth(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("threads create container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("threads create container: %s: %s", resp.Status, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("threads decode container: %w", err)
	}
	return result.ID, nil
}

// publishContainer finalizes a prepared container, turning it into a live post.
func (a *ThreadsAdapter) publishContainer(ctx context.Context, containerID string) error {
	endpoint := fmt.Sprintf("%s/%s/me/threads_publish", threadsBaseURL, threadsAPIVersion)

	values := url.Values{}
	values.Set("creation_id", containerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return fmt.Errorf("threads publish request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	a.setAuth(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("threads publish: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("threads publish: %s: %s", resp.Status, string(body))
	}
	return nil
}

func (a *ThreadsAdapter) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.accessToken)
}
