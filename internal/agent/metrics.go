package agent

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects agent performance and usage statistics.
type Metrics struct {
	// Request counters
	totalRequests  atomic.Int64
	totalErrors    atomic.Int64
	totalToolCalls atomic.Int64
	totalTokensIn  atomic.Int64
	totalTokensOut atomic.Int64

	// Latency tracking
	mu             sync.Mutex
	requestLatency []time.Duration
	maxLatency     time.Duration

	// Session counters
	activeSessions atomic.Int64
	totalSessions  atomic.Int64

	startTime time.Time
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:      time.Now(),
		requestLatency: make([]time.Duration, 0, 1000),
	}
}

// RecordRequest records a completed request with its latency.
func (m *Metrics) RecordRequest(latency time.Duration, err error) {
	m.totalRequests.Add(1)
	if err != nil {
		m.totalErrors.Add(1)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestLatency = append(m.requestLatency, latency)
	if latency > m.maxLatency {
		m.maxLatency = latency
	}
	// Keep only last 1000 latencies to bound memory
	if len(m.requestLatency) > 1000 {
		m.requestLatency = m.requestLatency[len(m.requestLatency)-1000:]
	}
}

// RecordToolCall increments the tool call counter.
func (m *Metrics) RecordToolCall() {
	m.totalToolCalls.Add(1)
}

// RecordTokens adds to the token counters.
func (m *Metrics) RecordTokens(in, out int64) {
	m.totalTokensIn.Add(in)
	m.totalTokensOut.Add(out)
}

// SessionStarted increments the active and total session counters.
func (m *Metrics) SessionStarted() {
	m.activeSessions.Add(1)
	m.totalSessions.Add(1)
}

// SessionEnded decrements the active session counter.
func (m *Metrics) SessionEnded() {
	m.activeSessions.Add(-1)
}

// MetricsSnapshot holds a point-in-time view of the metrics.
type MetricsSnapshot struct {
	Uptime         time.Duration `json:"uptime"`
	TotalRequests  int64         `json:"total_requests"`
	TotalErrors    int64         `json:"total_errors"`
	ErrorRate      float64       `json:"error_rate"`
	TotalToolCalls int64         `json:"total_tool_calls"`
	TotalTokensIn  int64         `json:"total_tokens_in"`
	TotalTokensOut int64         `json:"total_tokens_out"`
	ActiveSessions int64         `json:"active_sessions"`
	TotalSessions  int64         `json:"total_sessions"`
	AvgLatencyMs   float64       `json:"avg_latency_ms"`
	MaxLatencyMs   float64       `json:"max_latency_ms"`
	P95LatencyMs   float64       `json:"p95_latency_ms"`
}

// Snapshot returns a point-in-time copy of the current metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	totalReqs := m.totalRequests.Load()
	totalErrs := m.totalErrors.Load()

	var errorRate float64
	if totalReqs > 0 {
		errorRate = float64(totalErrs) / float64(totalReqs)
	}

	m.mu.Lock()
	avgMs := m.avgLatencyMsLocked()
	p95Ms := m.p95LatencyMsLocked()
	maxMs := float64(m.maxLatency.Milliseconds())
	m.mu.Unlock()

	return MetricsSnapshot{
		Uptime:         time.Since(m.startTime),
		TotalRequests:  totalReqs,
		TotalErrors:    totalErrs,
		ErrorRate:      errorRate,
		TotalToolCalls: m.totalToolCalls.Load(),
		TotalTokensIn:  m.totalTokensIn.Load(),
		TotalTokensOut: m.totalTokensOut.Load(),
		ActiveSessions: m.activeSessions.Load(),
		TotalSessions:  m.totalSessions.Load(),
		AvgLatencyMs:   avgMs,
		MaxLatencyMs:   maxMs,
		P95LatencyMs:   p95Ms,
	}
}

func (m *Metrics) avgLatencyMsLocked() float64 {
	if len(m.requestLatency) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range m.requestLatency {
		total += d
	}
	return float64(total.Milliseconds()) / float64(len(m.requestLatency))
}

func (m *Metrics) p95LatencyMsLocked() float64 {
	n := len(m.requestLatency)
	if n == 0 {
		return 0
	}
	// Simple P95: sort and pick the 95th percentile element.
	// We use insertion sort on a copy since the slice is bounded to 1000.
	sorted := make([]time.Duration, n)
	copy(sorted, m.requestLatency)
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}
	idx := int(float64(n) * 0.95)
	if idx >= n {
		idx = n - 1
	}
	return float64(sorted[idx].Milliseconds())
}
