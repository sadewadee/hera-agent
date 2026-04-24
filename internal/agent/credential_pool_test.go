package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- PooledCredential ---

func TestPooledCredential_RuntimeAPIKey_Standard(t *testing.T) {
	c := &PooledCredential{Provider: "openai", AccessToken: "sk-123"}
	assert.Equal(t, "sk-123", c.RuntimeAPIKey())
}

func TestPooledCredential_RuntimeAPIKey_Nous_AgentKey(t *testing.T) {
	c := &PooledCredential{Provider: "nous", AccessToken: "access", AgentKey: "agent-key"}
	assert.Equal(t, "agent-key", c.RuntimeAPIKey())
}

func TestPooledCredential_RuntimeAPIKey_Nous_NoAgentKey(t *testing.T) {
	c := &PooledCredential{Provider: "nous", AccessToken: "access"}
	assert.Equal(t, "access", c.RuntimeAPIKey())
}

func TestPooledCredential_RuntimeBaseURL_Standard(t *testing.T) {
	c := &PooledCredential{Provider: "openai", BaseURL: "https://api.openai.com"}
	assert.Equal(t, "https://api.openai.com", c.RuntimeBaseURL())
}

func TestPooledCredential_RuntimeBaseURL_Nous_InferenceURL(t *testing.T) {
	c := &PooledCredential{Provider: "nous", BaseURL: "https://base", InferenceBaseURL: "https://inference"}
	assert.Equal(t, "https://inference", c.RuntimeBaseURL())
}

func TestPooledCredential_RuntimeBaseURL_Nous_NoInferenceURL(t *testing.T) {
	c := &PooledCredential{Provider: "nous", BaseURL: "https://base"}
	assert.Equal(t, "https://base", c.RuntimeBaseURL())
}

// --- NewPooledCredential ---

func TestNewPooledCredential_Defaults(t *testing.T) {
	c := NewPooledCredential("openai", map[string]string{
		"access_token": "sk-test",
	})
	assert.Equal(t, "openai", c.Provider)
	assert.NotEmpty(t, c.ID)
	assert.Equal(t, "openai", c.Label)
	assert.Equal(t, AuthTypeAPIKey, c.AuthType)
	assert.Equal(t, SourceManual, c.Source)
	assert.Equal(t, "sk-test", c.AccessToken)
}

func TestNewPooledCredential_CustomValues(t *testing.T) {
	c := NewPooledCredential("anthropic", map[string]string{
		"access_token": "ak-test",
		"id":           "my-id",
		"label":        "my-label",
		"auth_type":    AuthTypeOAuth,
		"source":       "oauth-flow",
		"priority":     "5",
	})
	assert.Equal(t, "my-id", c.ID)
	assert.Equal(t, "my-label", c.Label)
	assert.Equal(t, AuthTypeOAuth, c.AuthType)
	assert.Equal(t, "oauth-flow", c.Source)
	assert.Equal(t, 5, c.Priority)
}

func TestNewPooledCredential_LabelFromSource(t *testing.T) {
	c := NewPooledCredential("openai", map[string]string{
		"access_token": "sk",
		"source":       "env-var",
	})
	assert.Equal(t, "env-var", c.Label)
}

// --- LabelFromToken ---

func TestLabelFromToken_Empty(t *testing.T) {
	assert.Equal(t, "fallback", LabelFromToken("", "fallback"))
}

func TestLabelFromToken_NonEmpty(t *testing.T) {
	assert.Equal(t, "fallback", LabelFromToken("some-token", "fallback"))
}

// --- NextPriority ---

func TestNextPriority_Empty(t *testing.T) {
	assert.Equal(t, 0, NextPriority(nil))
}

func TestNextPriority_WithEntries(t *testing.T) {
	entries := []*PooledCredential{
		{Priority: 2},
		{Priority: 5},
		{Priority: 3},
	}
	assert.Equal(t, 6, NextPriority(entries))
}

// --- IsManualSource ---

func TestIsManualSource(t *testing.T) {
	assert.True(t, IsManualSource("manual"))
	assert.True(t, IsManualSource("manual:env"))
	assert.True(t, IsManualSource("  Manual  "))
	assert.True(t, IsManualSource("MANUAL:prefix"))
	assert.False(t, IsManualSource("oauth"))
	assert.False(t, IsManualSource(""))
}

// --- ExhaustedTTL ---

func TestExhaustedTTL_429(t *testing.T) {
	assert.Equal(t, ExhaustedTTL429Seconds, ExhaustedTTL(429))
}

func TestExhaustedTTL_Other(t *testing.T) {
	assert.Equal(t, ExhaustedTTLDefaultSeconds, ExhaustedTTL(500))
	assert.Equal(t, ExhaustedTTLDefaultSeconds, ExhaustedTTL(0))
}

// --- ParseAbsoluteTimestamp ---

func TestParseAbsoluteTimestamp_Empty(t *testing.T) {
	_, ok := ParseAbsoluteTimestamp("")
	assert.False(t, ok)
}

func TestParseAbsoluteTimestamp_EpochSeconds(t *testing.T) {
	v, ok := ParseAbsoluteTimestamp("1700000000")
	assert.True(t, ok)
	assert.Equal(t, 1700000000.0, v)
}

func TestParseAbsoluteTimestamp_EpochMilliseconds(t *testing.T) {
	v, ok := ParseAbsoluteTimestamp("1700000000000")
	assert.True(t, ok)
	assert.Equal(t, 1700000000.0, v)
}

func TestParseAbsoluteTimestamp_NegativeValue(t *testing.T) {
	_, ok := ParseAbsoluteTimestamp("-1")
	assert.False(t, ok)
}

func TestParseAbsoluteTimestamp_Zero(t *testing.T) {
	_, ok := ParseAbsoluteTimestamp("0")
	assert.False(t, ok)
}

func TestParseAbsoluteTimestamp_ISO8601(t *testing.T) {
	v, ok := ParseAbsoluteTimestamp("2024-01-01T00:00:00Z")
	assert.True(t, ok)
	assert.Greater(t, v, 0.0)
}

func TestParseAbsoluteTimestamp_ISO8601_NoZ(t *testing.T) {
	v, ok := ParseAbsoluteTimestamp("2024-01-01T00:00:00+00:00")
	assert.True(t, ok)
	assert.Greater(t, v, 0.0)
}

func TestParseAbsoluteTimestamp_Invalid(t *testing.T) {
	_, ok := ParseAbsoluteTimestamp("not-a-timestamp")
	assert.False(t, ok)
}

// --- ExtractRetryDelaySeconds ---

func TestExtractRetryDelaySeconds_Empty(t *testing.T) {
	_, ok := ExtractRetryDelaySeconds("")
	assert.False(t, ok)
}

func TestExtractRetryDelaySeconds_QuotaResetDelay_MS(t *testing.T) {
	v, ok := ExtractRetryDelaySeconds(`quotaResetDelay: "5000ms"`)
	assert.True(t, ok)
	assert.Equal(t, 5.0, v)
}

func TestExtractRetryDelaySeconds_QuotaResetDelay_S(t *testing.T) {
	v, ok := ExtractRetryDelaySeconds(`quotaResetDelay: "30s"`)
	assert.True(t, ok)
	assert.Equal(t, 30.0, v)
}

func TestExtractRetryDelaySeconds_RetryAfter(t *testing.T) {
	v, ok := ExtractRetryDelaySeconds("Please retry after 60 seconds")
	assert.True(t, ok)
	assert.Equal(t, 60.0, v)
}

func TestExtractRetryDelaySeconds_NoMatch(t *testing.T) {
	_, ok := ExtractRetryDelaySeconds("some random error message")
	assert.False(t, ok)
}

// --- NormalizeErrorContext ---

func TestNormalizeErrorContext_Nil(t *testing.T) {
	result := NormalizeErrorContext(nil)
	assert.Empty(t, result)
}

func TestNormalizeErrorContext_Empty(t *testing.T) {
	result := NormalizeErrorContext(map[string]string{})
	assert.Empty(t, result)
}

func TestNormalizeErrorContext_ExtractsKnownFields(t *testing.T) {
	ctx := map[string]string{
		"reason":      " rate_limited ",
		"message":     " Too many requests ",
		"reset_at":    "2024-01-01T00:00:00Z",
		"retry_after": "60",
		"unknown":     "ignored",
	}
	result := NormalizeErrorContext(ctx)
	assert.Equal(t, "rate_limited", result["reason"])
	assert.Equal(t, "Too many requests", result["message"])
	assert.Equal(t, "2024-01-01T00:00:00Z", result["reset_at"])
	assert.Equal(t, "60", result["retry_after"])
	_, ok := result["unknown"]
	assert.False(t, ok)
}

func TestNormalizeErrorContext_SkipsEmptyValues(t *testing.T) {
	ctx := map[string]string{
		"reason":  "  ",
		"message": "",
	}
	result := NormalizeErrorContext(ctx)
	assert.Empty(t, result)
}

// --- CredentialPool ---

func TestNewCredentialPool_DefaultStrategy(t *testing.T) {
	pool := NewCredentialPool("openai", "invalid-strategy")
	assert.Equal(t, StrategyRoundRobin, pool.strategy)
}

func TestNewCredentialPool_ValidStrategy(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyLeastUsed)
	assert.Equal(t, StrategyLeastUsed, pool.strategy)
}

func TestCredentialPool_AddAndSize(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyRoundRobin)
	assert.Equal(t, 0, pool.Size())
	pool.Add(&PooledCredential{ID: "a", AccessToken: "sk-a"})
	pool.Add(&PooledCredential{ID: "b", AccessToken: "sk-b"})
	assert.Equal(t, 2, pool.Size())
}

func TestCredentialPool_Next_Empty(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyRoundRobin)
	assert.Nil(t, pool.Next())
}

func TestCredentialPool_RoundRobin(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyRoundRobin)
	pool.Add(&PooledCredential{ID: "a", AccessToken: "sk-a"})
	pool.Add(&PooledCredential{ID: "b", AccessToken: "sk-b"})
	pool.Add(&PooledCredential{ID: "c", AccessToken: "sk-c"})

	ids := make([]string, 6)
	for i := 0; i < 6; i++ {
		cred := pool.Next()
		require.NotNil(t, cred)
		ids[i] = cred.ID
	}
	assert.Equal(t, "a", ids[0])
	assert.Equal(t, "b", ids[1])
	assert.Equal(t, "c", ids[2])
	assert.Equal(t, "a", ids[3])
}

func TestCredentialPool_FillFirst(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyFillFirst)
	pool.Add(&PooledCredential{ID: "a", AccessToken: "sk-a"})
	pool.Add(&PooledCredential{ID: "b", AccessToken: "sk-b"})

	// Should always return the first one.
	for i := 0; i < 5; i++ {
		cred := pool.Next()
		require.NotNil(t, cred)
		assert.Equal(t, "a", cred.ID)
	}
}

func TestCredentialPool_LeastUsed(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyLeastUsed)
	pool.Add(&PooledCredential{ID: "a", AccessToken: "sk-a", RequestCount: 100})
	pool.Add(&PooledCredential{ID: "b", AccessToken: "sk-b", RequestCount: 10})
	pool.Add(&PooledCredential{ID: "c", AccessToken: "sk-c", RequestCount: 50})

	cred := pool.Next()
	require.NotNil(t, cred)
	assert.Equal(t, "b", cred.ID)
}

func TestCredentialPool_MarkExhausted_SkipsKey(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyRoundRobin)
	pool.Add(&PooledCredential{ID: "a", AccessToken: "sk-a"})
	pool.Add(&PooledCredential{ID: "b", AccessToken: "sk-b"})

	pool.MarkExhausted("a", 10*time.Minute)

	cred := pool.Next()
	require.NotNil(t, cred)
	assert.Equal(t, "b", cred.ID)
}

func TestCredentialPool_AllExhausted_ReturnsSoonest(t *testing.T) {
	pool := NewCredentialPool("openai", StrategyRoundRobin)
	pool.Add(&PooledCredential{ID: "a", AccessToken: "sk-a"})
	pool.Add(&PooledCredential{ID: "b", AccessToken: "sk-b"})

	pool.MarkExhausted("a", 10*time.Minute)
	pool.MarkExhausted("b", 30*time.Minute)

	cred := pool.Next()
	require.NotNil(t, cred)
	// "a" expires sooner, so it should be returned.
	assert.Equal(t, "a", cred.ID)
}

// --- GetCustomProviderPoolKey ---

func TestGetCustomProviderPoolKey(t *testing.T) {
	assert.Equal(t, "custom:my-provider", GetCustomProviderPoolKey("My Provider"))
	assert.Equal(t, "custom:test", GetCustomProviderPoolKey("  Test  "))
}

// --- SupportedPoolStrategies ---

func TestSupportedPoolStrategies(t *testing.T) {
	assert.True(t, SupportedPoolStrategies[StrategyFillFirst])
	assert.True(t, SupportedPoolStrategies[StrategyRoundRobin])
	assert.True(t, SupportedPoolStrategies[StrategyRandom])
	assert.True(t, SupportedPoolStrategies[StrategyLeastUsed])
	assert.False(t, SupportedPoolStrategies["nonexistent"])
}
