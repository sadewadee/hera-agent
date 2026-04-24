package agent

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimitTracker(t *testing.T) {
	tracker := NewRateLimitTracker()
	require.NotNil(t, tracker)
	assert.NotNil(t, tracker.tracking)
}

func TestTrackResponse_NilResponse(t *testing.T) {
	tracker := NewRateLimitTracker()
	limited := tracker.TrackResponse("openai", "key1", nil)
	assert.False(t, limited)
}

func TestTrackResponse_200OK(t *testing.T) {
	tracker := NewRateLimitTracker()
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
	}
	limited := tracker.TrackResponse("openai", "key1", resp)
	assert.False(t, limited)
	assert.False(t, tracker.IsLimited("openai", "key1"))
}

func TestTrackResponse_429RateLimited(t *testing.T) {
	tracker := NewRateLimitTracker()
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
	}
	limited := tracker.TrackResponse("openai", "key1", resp)
	assert.True(t, limited)
	assert.True(t, tracker.IsLimited("openai", "key1"))
}

func TestTrackResponse_429WithRetryAfterSeconds(t *testing.T) {
	tracker := NewRateLimitTracker()
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"5"}},
	}
	tracker.TrackResponse("openai", "key1", resp)
	backoff := tracker.GetBackoff("openai", "key1")
	// Should be roughly 5 seconds (allow for timing).
	assert.InDelta(t, 5.0, backoff.Seconds(), 1.0)
}

func TestTrackResponse_ParsesRateLimitHeaders(t *testing.T) {
	tracker := NewRateLimitTracker()
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"X-Ratelimit-Remaining-Requests": []string{"42"},
			"X-Ratelimit-Remaining-Tokens":   []string{"1000"},
		},
	}
	tracker.TrackResponse("anthropic", "key2", resp)
	info := tracker.GetInfo("anthropic", "key2")
	require.NotNil(t, info)
	assert.Equal(t, 42, info.RequestsLeft)
	assert.Equal(t, 1000, info.TokensLeft)
	assert.False(t, info.Limited)
}

func TestIsLimited_Unknown(t *testing.T) {
	tracker := NewRateLimitTracker()
	assert.False(t, tracker.IsLimited("nope", "nope"))
}

func TestIsLimited_ExpiresAfterBackoff(t *testing.T) {
	tracker := NewRateLimitTracker()
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"0"}},
	}
	tracker.TrackResponse("openai", "key1", resp)
	// With 0-second retry, should clear immediately.
	time.Sleep(10 * time.Millisecond)
	assert.False(t, tracker.IsLimited("openai", "key1"))
}

func TestGetBackoff_NotLimited(t *testing.T) {
	tracker := NewRateLimitTracker()
	assert.Equal(t, time.Duration(0), tracker.GetBackoff("openai", "key1"))
}

func TestGetInfo_NotTracked(t *testing.T) {
	tracker := NewRateLimitTracker()
	assert.Nil(t, tracker.GetInfo("openai", "key1"))
}

func TestGetInfo_ReturnsCopy(t *testing.T) {
	tracker := NewRateLimitTracker()
	resp := &http.Response{StatusCode: 200, Header: http.Header{}}
	tracker.TrackResponse("openai", "key1", resp)
	info := tracker.GetInfo("openai", "key1")
	require.NotNil(t, info)
	info.RequestsLeft = 999
	info2 := tracker.GetInfo("openai", "key1")
	assert.NotEqual(t, 999, info2.RequestsLeft)
}

func TestParseRetryAfter_Empty(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	d := parseRetryAfter(resp)
	assert.Equal(t, 60*time.Second, d)
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"30"}}}
	d := parseRetryAfter(resp)
	assert.Equal(t, 30*time.Second, d)
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	future := time.Now().Add(120 * time.Second).UTC().Format(time.RFC1123)
	resp := &http.Response{Header: http.Header{"Retry-After": []string{future}}}
	d := parseRetryAfter(resp)
	assert.InDelta(t, 120.0, d.Seconds(), 2.0)
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"not-valid"}}}
	d := parseRetryAfter(resp)
	assert.Equal(t, 60*time.Second, d)
}

func TestTrackResponse_ClearsLimitOn200(t *testing.T) {
	tracker := NewRateLimitTracker()
	resp429 := &http.Response{StatusCode: 429, Header: http.Header{}}
	tracker.TrackResponse("openai", "key1", resp429)
	assert.True(t, tracker.IsLimited("openai", "key1"))

	resp200 := &http.Response{StatusCode: 200, Header: http.Header{}}
	tracker.TrackResponse("openai", "key1", resp200)
	// After 200 response, Limited flag is cleared; but RetryAfter may still
	// be in the future. The 200 sets Limited=false.
	info := tracker.GetInfo("openai", "key1")
	assert.False(t, info.Limited)
}
