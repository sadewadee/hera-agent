package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// GatewayOptions configures the gateway orchestrator.
type GatewayOptions struct {
	SessionTimeout    time.Duration // idle session expiry
	HealthInterval    time.Duration // adapter health check interval
	ReconnectBaseWait time.Duration // initial reconnection delay
	ReconnectMaxWait  time.Duration // cap on exponential backoff
	MaxReconnects     int           // max consecutive reconnection attempts
}

func (o *GatewayOptions) withDefaults() {
	if o.SessionTimeout == 0 {
		o.SessionTimeout = 30 * time.Minute
	}
	if o.HealthInterval == 0 {
		o.HealthInterval = 30 * time.Second
	}
	if o.ReconnectBaseWait == 0 {
		o.ReconnectBaseWait = 1 * time.Second
	}
	if o.ReconnectMaxWait == 0 {
		o.ReconnectMaxWait = 5 * time.Minute
	}
	if o.MaxReconnects == 0 {
		o.MaxReconnects = 10
	}
}

// Gateway manages all platform adapters, routes incoming messages, and
// handles reconnections with exponential backoff.
type Gateway struct {
	mu           sync.RWMutex
	opts         GatewayOptions
	adapters     []PlatformAdapter
	sessions     *SessionManager
	router       *Router
	hooks        *HookManager
	mirror       *MirrorManager
	stickerCache *StickerCache
	channels     *ChannelDirectory
	pairing      *PairingStore
	allowAll     bool
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	logger       *slog.Logger
}

// SetAllowAll skips the per-user authorization check when true. Useful
// for single-user deployments and development.
func (g *Gateway) SetAllowAll(v bool) {
	g.mu.Lock()
	g.allowAll = v
	g.mu.Unlock()
}

// PreAuthorize seeds the pairing store from config allow_list entries so
// the user can chat with the bot immediately without the /pair dance.
func (g *Gateway) PreAuthorize(platform string, userIDs ...string) {
	for _, u := range userIDs {
		g.pairing.Authorize(platform, u)
	}
}

// NewGateway creates a new gateway with the provided options.
func NewGateway(opts GatewayOptions) *Gateway {
	opts.withDefaults()
	sm := NewSessionManager(opts.SessionTimeout)
	r := NewRouter(sm)

	return &Gateway{
		opts:         opts,
		adapters:     make([]PlatformAdapter, 0),
		sessions:     sm,
		router:       r,
		hooks:        NewHookManager(),
		stickerCache: NewStickerCache(500, 10*time.Minute),
		channels:     NewChannelDirectory(),
		pairing:      NewPairingStore(24 * time.Hour),
		logger:       slog.Default(),
	}
}

// AddAdapter registers a platform adapter with the gateway.
func (g *Gateway) AddAdapter(a PlatformAdapter) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.adapters = append(g.adapters, a)
}

// Adapters returns a snapshot of the registered adapters.
func (g *Gateway) Adapters() []PlatformAdapter {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]PlatformAdapter, len(g.adapters))
	copy(out, g.adapters)
	return out
}

// OnMessage sets the handler that receives routed messages.
func (g *Gateway) OnMessage(h RouteHandler) {
	g.router.SetHandler(h)
}

// Start connects all adapters in goroutines and begins health checking.
func (g *Gateway) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	g.cancel = cancel

	// Wire each adapter's message callback through the router.
	g.mu.RLock()
	adapters := make([]PlatformAdapter, len(g.adapters))
	copy(adapters, g.adapters)
	g.mu.RUnlock()

	for _, a := range adapters {
		adapter := a
		adapter.OnMessage(func(mctx context.Context, msg IncomingMessage) {
			// Wire: channel directory registration
			g.channels.Register(ChannelEntry{
				Platform: msg.Platform,
				ChatID:   msg.ChatID,
				Active:   true,
			})

			// Wire: pairing authorization check. allow_all short-circuits
			// for single-user / dev deployments; otherwise the user must
			// be in the seeded allow_list or have paired via /pair.
			g.mu.RLock()
			allowAll := g.allowAll
			g.mu.RUnlock()
			if !allowAll && !g.pairing.IsAuthorized(msg.Platform, msg.UserID) {
				g.logger.Info("unauthorized message, dropping",
					"platform", msg.Platform,
					"user", msg.UserID,
					"hint", "add user to gateway.platforms."+msg.Platform+".allow_list or enable gateway.allow_all",
				)
				return
			}

			// Wire: hooks before message
			modified, err := g.hooks.RunBefore(mctx, &msg)
			if err != nil {
				g.logger.Warn("hook before error", "error", err)
				return
			}
			if modified != nil {
				msg = *modified
			}

			// Route the message
			g.router.Route(mctx, msg)

			// Wire: hooks after message (fire and forget, response not captured in async flow)
			_ = g.hooks.RunAfter(mctx, &msg, "")

			// Wire: mirror forwarding
			if g.mirror != nil {
				_ = g.mirror.ProcessMessage(mctx, msg.Platform, msg.ChatID, msg.Text)
			}
		})
		g.wg.Add(1)
		go g.runAdapter(ctx, adapter)
	}

	// Start session expiry watcher.
	g.sessions.StartExpiryWatcher(ctx, g.opts.HealthInterval)

	// Start health check loop.
	g.wg.Add(1)
	go g.healthLoop(ctx, adapters)

	return nil
}

// Stop gracefully shuts down all adapters and cancels background work.
func (g *Gateway) Stop() {
	if g.cancel != nil {
		g.cancel()
	}
	g.wg.Wait()

	g.mu.RLock()
	adapters := make([]PlatformAdapter, len(g.adapters))
	copy(adapters, g.adapters)
	g.mu.RUnlock()

	for _, a := range adapters {
		if a.IsConnected() {
			if err := a.Disconnect(context.Background()); err != nil {
				g.logger.Error("disconnect adapter", "adapter", a.Name(), "error", err)
			}
		}
	}
}

// Sessions returns the session manager for external inspection.
func (g *Gateway) Sessions() *SessionManager {
	return g.sessions
}

// Hooks returns the hook chain for adding message hooks.
func (g *Gateway) Hooks() *HookManager { return g.hooks }

// StickerCache returns the sticker cache.
func (g *Gateway) StickerCache() *StickerCache { return g.stickerCache }

// Channels returns the channel directory.
func (g *Gateway) Channels() *ChannelDirectory { return g.channels }

// Pairing returns the pairing store for authorization management.
func (g *Gateway) Pairing() *PairingStore { return g.pairing }

// SetMirror sets the mirror manager for message forwarding.
func (g *Gateway) SetMirror(m *MirrorManager) { g.mirror = m }

// Mirror returns the mirror manager (may be nil).
func (g *Gateway) Mirror() *MirrorManager { return g.mirror }

// FindAdapter returns the first adapter whose Name() matches the given platform,
// or nil if none is found.
func (g *Gateway) FindAdapter(platform string) PlatformAdapter {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, a := range g.adapters {
		if a.Name() == platform {
			return a
		}
	}
	return nil
}

// runAdapter connects a single adapter with reconnection backoff.
func (g *Gateway) runAdapter(ctx context.Context, a PlatformAdapter) {
	defer g.wg.Done()
	bo := newBackoff(g.opts.ReconnectBaseWait, g.opts.ReconnectMaxWait, g.opts.MaxReconnects)

	for {
		err := a.Connect(ctx)
		if err == nil {
			g.logger.Info("adapter connected", "adapter", a.Name())
			bo.Reset()
			// Stay alive until context is cancelled or adapter disconnects.
			g.waitUntilDisconnected(ctx, a)
		} else {
			g.logger.Warn("adapter connect failed", "adapter", a.Name(), "error", err)
		}

		// Check if we should stop entirely.
		select {
		case <-ctx.Done():
			return
		default:
		}

		if bo.Exhausted() {
			g.logger.Error("max reconnection attempts reached", "adapter", a.Name())
			return
		}

		wait := bo.Next()
		g.logger.Info("reconnecting adapter", "adapter", a.Name(), "wait", wait)

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// waitUntilDisconnected blocks until the adapter reports disconnected or ctx is cancelled.
func (g *Gateway) waitUntilDisconnected(ctx context.Context, a PlatformAdapter) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !a.IsConnected() {
				return
			}
		}
	}
}

// healthLoop periodically checks adapter connectivity.
func (g *Gateway) healthLoop(ctx context.Context, adapters []PlatformAdapter) {
	defer g.wg.Done()
	ticker := time.NewTicker(g.opts.HealthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, a := range adapters {
				if !a.IsConnected() {
					g.logger.Warn("adapter unhealthy", "adapter", a.Name())
				}
			}
		}
	}
}

// SendTo sends an outgoing message to a specific adapter and chat.
func (g *Gateway) SendTo(ctx context.Context, platform, chatID string, msg OutgoingMessage) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, a := range g.adapters {
		if a.Name() == platform {
			if !a.IsConnected() {
				return fmt.Errorf("adapter %q is not connected", platform)
			}
			return a.Send(ctx, chatID, msg)
		}
	}
	return fmt.Errorf("no adapter found for platform %q", platform)
}

// --- backoff ---

// backoff implements truncated exponential backoff with a retry counter.
type backoff struct {
	base     time.Duration
	max      time.Duration
	maxTries int
	attempt  int
}

func newBackoff(base, max time.Duration, maxTries int) *backoff {
	return &backoff{base: base, max: max, maxTries: maxTries}
}

// Next returns the next wait duration and advances the attempt counter.
func (b *backoff) Next() time.Duration {
	d := b.base << uint(b.attempt)
	if d > b.max {
		d = b.max
	}
	b.attempt++
	return d
}

// Exhausted returns true when max retries have been consumed.
func (b *backoff) Exhausted() bool {
	return b.attempt >= b.maxTries
}

// Reset clears the attempt counter.
func (b *backoff) Reset() {
	b.attempt = 0
}
