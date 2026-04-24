// Package platforms provides platform adapter implementations for the gateway.
//
// telegram_network.go provides a hostname-preserving fallback transport for
// networks where api.telegram.org is unreachable. It resolves fallback IPs
// via DNS-over-HTTPS (Google and Cloudflare) and retries TCP connections
// against known-reachable IPs while preserving TLS SNI.
package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	telegramAPIHost = "api.telegram.org"
	dohTimeout      = 4 * time.Second
)

// DOH providers for discovering Telegram API IPs.
var dohProviders = []dohProvider{
	{
		URL:     "https://dns.google/resolve",
		Params:  map[string]string{"name": telegramAPIHost, "type": "A"},
		Headers: nil,
	},
	{
		URL:     "https://cloudflare-dns.com/dns-query",
		Params:  map[string]string{"name": telegramAPIHost, "type": "A"},
		Headers: map[string]string{"Accept": "application/dns-json"},
	},
}

// Last-resort IPs in the 149.154.160.0/20 block.
var seedFallbackIPs = []string{"149.154.167.220"}

type dohProvider struct {
	URL     string
	Params  map[string]string
	Headers map[string]string
}

// TelegramFallbackTransport wraps http.RoundTripper to retry Telegram
// Bot API requests via fallback IPs while preserving TLS/SNI.
type TelegramFallbackTransport struct {
	mu          sync.Mutex
	fallbackIPs []string
	primary     http.RoundTripper
	stickyIP    string
}

// NewTelegramFallbackTransport creates a transport with fallback IPs.
func NewTelegramFallbackTransport(fallbackIPs []string, base http.RoundTripper) *TelegramFallbackTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	normalized := NormalizeFallbackIPs(fallbackIPs)
	return &TelegramFallbackTransport{
		fallbackIPs: normalized,
		primary:     base,
	}
}

// RoundTrip implements http.RoundTripper with fallback IP retry logic.
func (t *TelegramFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != telegramAPIHost || len(t.fallbackIPs) == 0 {
		return t.primary.RoundTrip(req)
	}

	t.mu.Lock()
	stickyIP := t.stickyIP
	t.mu.Unlock()

	// Build attempt order: sticky IP first (if set), then remaining fallbacks.
	attempts := make([]string, 0, len(t.fallbackIPs)+1)
	if stickyIP != "" {
		attempts = append(attempts, stickyIP)
	} else {
		attempts = append(attempts, "") // empty = try primary
	}
	for _, ip := range t.fallbackIPs {
		if ip != stickyIP {
			attempts = append(attempts, ip)
		}
	}

	var lastErr error
	for _, ip := range attempts {
		var resp *http.Response
		var err error

		if ip == "" {
			resp, err = t.primary.RoundTrip(req)
		} else {
			// Rewrite request to use fallback IP.
			clone := req.Clone(req.Context())
			clone.URL.Host = ip
			clone.Host = telegramAPIHost
			resp, err = t.primary.RoundTrip(clone)
		}

		if err == nil {
			if ip != "" && ip != stickyIP {
				t.mu.Lock()
				t.stickyIP = ip
				t.mu.Unlock()
				slog.Warn("Telegram primary path unreachable, using fallback IP",
					"ip", ip,
				)
			}
			return resp, nil
		}

		lastErr = err
		if !isRetryableConnectError(err) {
			return nil, err
		}

		if ip == "" {
			slog.Warn("Telegram primary connection failed, trying fallbacks",
				"error", err,
				"fallback_ips", strings.Join(t.fallbackIPs, ", "),
			)
		} else {
			slog.Warn("Telegram fallback IP failed",
				"ip", ip,
				"error", err,
			)
		}
	}

	if lastErr == nil {
		return nil, fmt.Errorf("all Telegram fallback IPs exhausted")
	}
	return nil, lastErr
}

// NormalizeFallbackIPs validates and normalizes a list of fallback IPs.
func NormalizeFallbackIPs(values []string) []string {
	var normalized []string
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		ip := net.ParseIP(raw)
		if ip == nil {
			slog.Warn("Ignoring invalid Telegram fallback IP", "ip", raw)
			continue
		}
		if ip.To4() == nil {
			slog.Warn("Ignoring non-IPv4 Telegram fallback IP", "ip", raw)
			continue
		}
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			slog.Warn("Ignoring private/internal Telegram fallback IP", "ip", raw)
			continue
		}
		normalized = append(normalized, ip.String())
	}
	return normalized
}

// ParseFallbackIPEnv parses a comma-separated list of fallback IPs from
// an environment variable value.
func ParseFallbackIPEnv(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			trimmed = append(trimmed, p)
		}
	}
	return NormalizeFallbackIPs(trimmed)
}

// DiscoverFallbackIPs auto-discovers Telegram API IPs via DNS-over-HTTPS.
func DiscoverFallbackIPs(ctx context.Context) []string {
	client := &http.Client{Timeout: dohTimeout}

	// Resolve system DNS IPs.
	systemIPs := resolveSystemDNS()

	// Query DoH providers concurrently.
	type result struct {
		ips []string
		err error
	}
	ch := make(chan result, len(dohProviders))
	for _, provider := range dohProviders {
		go func(p dohProvider) {
			ips, err := queryDOHProvider(ctx, client, p)
			ch <- result{ips: ips, err: err}
		}(provider)
	}

	var dohIPs []string
	for range dohProviders {
		r := <-ch
		if r.err == nil {
			dohIPs = append(dohIPs, r.ips...)
		}
	}

	// Deduplicate and exclude system DNS IPs.
	seen := make(map[string]bool)
	var candidates []string
	for _, ip := range dohIPs {
		if !seen[ip] && !systemIPs[ip] {
			seen[ip] = true
			candidates = append(candidates, ip)
		}
	}

	validated := NormalizeFallbackIPs(candidates)
	if len(validated) > 0 {
		slog.Debug("Discovered Telegram fallback IPs via DoH",
			"ips", strings.Join(validated, ", "),
		)
		return validated
	}

	slog.Info("DoH discovery yielded no new IPs, using seed fallback IPs",
		"seed_ips", strings.Join(seedFallbackIPs, ", "),
	)
	return append([]string{}, seedFallbackIPs...)
}

func resolveSystemDNS() map[string]bool {
	ips := make(map[string]bool)
	addrs, err := net.LookupHost(telegramAPIHost)
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		ips[addr] = true
	}
	return ips
}

type dohResponse struct {
	Answer []dohAnswer `json:"Answer"`
}

type dohAnswer struct {
	Type int    `json:"type"`
	Data string `json:"data"`
}

func queryDOHProvider(ctx context.Context, client *http.Client, provider dohProvider) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.URL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for k, v := range provider.Params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	for k, v := range provider.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data dohResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var ips []string
	for _, answer := range data.Answer {
		if answer.Type != 1 { // A record
			continue
		}
		raw := strings.TrimSpace(answer.Data)
		if net.ParseIP(raw) != nil {
			ips = append(ips, raw)
		}
	}
	return ips, nil
}

func isRetryableConnectError(err error) bool {
	if err == nil {
		return false
	}
	// Check for net.OpError with dial/connect failures.
	var opErr *net.OpError
	if ok := isOpError(err, &opErr); ok {
		return opErr.Op == "dial" || opErr.Op == "connect"
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection timed out") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "i/o timeout")
}

func isOpError(err error, target **net.OpError) bool {
	for err != nil {
		if opErr, ok := err.(*net.OpError); ok {
			*target = opErr
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			return false
		}
	}
	return false
}
