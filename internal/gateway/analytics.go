package gateway

import (
	"sync"
	"time"
)

// MessageStats tracks statistics for a single platform or user.
type MessageStats struct {
	TotalMessages  int64     `json:"total_messages"`
	TotalResponses int64     `json:"total_responses"`
	LastMessageAt  time.Time `json:"last_message_at"`
	AvgResponseMs  float64   `json:"avg_response_ms"`
}

// Analytics tracks message analytics across the gateway.
type Analytics struct {
	mu             sync.Mutex
	byPlatform     map[string]*MessageStats
	byUser         map[string]*MessageStats
	totalMessages  int64
	totalResponses int64
	responseTimes  []time.Duration
	startTime      time.Time
}

// NewAnalytics creates a new analytics tracker.
func NewAnalytics() *Analytics {
	return &Analytics{
		byPlatform:    make(map[string]*MessageStats),
		byUser:        make(map[string]*MessageStats),
		responseTimes: make([]time.Duration, 0, 1000),
		startTime:     time.Now(),
	}
}

// RecordMessage records an incoming message for analytics.
func (a *Analytics) RecordMessage(platform, userID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.totalMessages++
	now := time.Now()

	ps := a.getOrCreatePlatform(platform)
	ps.TotalMessages++
	ps.LastMessageAt = now

	us := a.getOrCreateUser(userID)
	us.TotalMessages++
	us.LastMessageAt = now
}

// RecordResponse records a response with its latency.
func (a *Analytics) RecordResponse(platform, userID string, latency time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.totalResponses++
	a.responseTimes = append(a.responseTimes, latency)
	// Bound memory
	if len(a.responseTimes) > 10000 {
		a.responseTimes = a.responseTimes[len(a.responseTimes)-10000:]
	}

	ps := a.getOrCreatePlatform(platform)
	ps.TotalResponses++
	ps.AvgResponseMs = updateAvg(ps.AvgResponseMs, ps.TotalResponses, latency)

	us := a.getOrCreateUser(userID)
	us.TotalResponses++
	us.AvgResponseMs = updateAvg(us.AvgResponseMs, us.TotalResponses, latency)
}

// AnalyticsSnapshot holds a summary of gateway analytics.
type AnalyticsSnapshot struct {
	Uptime          time.Duration            `json:"uptime"`
	TotalMessages   int64                    `json:"total_messages"`
	TotalResponses  int64                    `json:"total_responses"`
	AvgResponseMs   float64                  `json:"avg_response_ms"`
	PlatformStats   map[string]*MessageStats `json:"platform_stats"`
	ActivePlatforms int                      `json:"active_platforms"`
	ActiveUsers     int                      `json:"active_users"`
}

// Snapshot returns a point-in-time view of the analytics.
func (a *Analytics) Snapshot() AnalyticsSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	platformCopy := make(map[string]*MessageStats, len(a.byPlatform))
	for k, v := range a.byPlatform {
		cp := *v
		platformCopy[k] = &cp
	}

	var avgMs float64
	if len(a.responseTimes) > 0 {
		var total time.Duration
		for _, d := range a.responseTimes {
			total += d
		}
		avgMs = float64(total.Milliseconds()) / float64(len(a.responseTimes))
	}

	return AnalyticsSnapshot{
		Uptime:          time.Since(a.startTime),
		TotalMessages:   a.totalMessages,
		TotalResponses:  a.totalResponses,
		AvgResponseMs:   avgMs,
		PlatformStats:   platformCopy,
		ActivePlatforms: len(a.byPlatform),
		ActiveUsers:     len(a.byUser),
	}
}

func (a *Analytics) getOrCreatePlatform(platform string) *MessageStats {
	stats, ok := a.byPlatform[platform]
	if !ok {
		stats = &MessageStats{}
		a.byPlatform[platform] = stats
	}
	return stats
}

func (a *Analytics) getOrCreateUser(userID string) *MessageStats {
	stats, ok := a.byUser[userID]
	if !ok {
		stats = &MessageStats{}
		a.byUser[userID] = stats
	}
	return stats
}

func updateAvg(currentAvg float64, count int64, newValue time.Duration) float64 {
	ms := float64(newValue.Milliseconds())
	return currentAvg + (ms-currentAvg)/float64(count)
}
