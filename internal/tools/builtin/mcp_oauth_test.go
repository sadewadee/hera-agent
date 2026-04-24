package builtin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOAuthTokenStore(t *testing.T) {
	dir := t.TempDir()
	store := NewOAuthTokenStore(dir)
	require.NotNil(t, store)
	assert.Equal(t, filepath.Join(dir, "mcp_oauth_tokens.json"), store.path)
}

func TestOAuthTokenStore_SetAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewOAuthTokenStore(dir)

	token := &OAuthToken{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	store.Set("server1", token)

	got := store.Get("server1")
	require.NotNil(t, got)
	assert.Equal(t, "test-access", got.AccessToken)
	assert.Equal(t, "test-refresh", got.RefreshToken)
}

func TestOAuthTokenStore_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewOAuthTokenStore(dir)
	got := store.Get("nonexistent")
	assert.Nil(t, got)
}

func TestOAuthTokenStore_GetExpired(t *testing.T) {
	dir := t.TempDir()
	store := NewOAuthTokenStore(dir)

	token := &OAuthToken{
		AccessToken: "expired",
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
	}
	store.Set("server1", token)

	got := store.Get("server1")
	assert.Nil(t, got)
}

func TestOAuthTokenStore_GetNoExpiry(t *testing.T) {
	dir := t.TempDir()
	store := NewOAuthTokenStore(dir)
	token := &OAuthToken{AccessToken: "no-expiry"}
	store.Set("server1", token)
	got := store.Get("server1")
	require.NotNil(t, got)
	assert.Equal(t, "no-expiry", got.AccessToken)
}

func TestOAuthTokenStore_Remove(t *testing.T) {
	dir := t.TempDir()
	store := NewOAuthTokenStore(dir)
	store.Set("s1", &OAuthToken{AccessToken: "t1"})
	store.Remove("s1")
	assert.Nil(t, store.Get("s1"))
}

func TestOAuthTokenStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	store := NewOAuthTokenStore(dir)
	store.Set("srv", &OAuthToken{AccessToken: "persist-token", ExpiresAt: time.Now().Add(time.Hour)})

	// Load from disk
	store2 := NewOAuthTokenStore(dir)
	got := store2.Get("srv")
	require.NotNil(t, got)
	assert.Equal(t, "persist-token", got.AccessToken)
}

func TestGenerateCodeVerifier(t *testing.T) {
	v1 := GenerateCodeVerifier()
	v2 := GenerateCodeVerifier()
	assert.NotEmpty(t, v1)
	assert.NotEmpty(t, v2)
	assert.NotEqual(t, v1, v2)
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(verifier)
	assert.NotEmpty(t, challenge)
	// Same verifier should produce same challenge
	assert.Equal(t, challenge, GenerateCodeChallenge(verifier))
}

func TestExchangeAuthCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "read write",
		})
	}))
	defer srv.Close()

	token, err := ExchangeAuthCode(srv.URL, "auth-code", "http://localhost/callback", "client-id", "verifier")
	require.NoError(t, err)
	assert.Equal(t, "new-access", token.AccessToken)
	assert.Equal(t, "new-refresh", token.RefreshToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.False(t, token.ExpiresAt.IsZero())
}

func TestExchangeAuthCode_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer srv.Close()

	_, err := ExchangeAuthCode(srv.URL, "bad-code", "http://localhost/callback", "client-id", "verifier")
	assert.Error(t, err)
}

func TestRefreshAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "refreshed-access",
			"refresh_token": "new-refresh",
			"token_type":    "Bearer",
			"expires_in":    7200,
		})
	}))
	defer srv.Close()

	token, err := RefreshAccessToken(srv.URL, "old-refresh", "client-id")
	require.NoError(t, err)
	assert.Equal(t, "refreshed-access", token.AccessToken)
	assert.Equal(t, "new-refresh", token.RefreshToken)
}

func TestRefreshAccessToken_KeepsOldRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "refreshed",
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	token, err := RefreshAccessToken(srv.URL, "old-refresh", "client-id")
	require.NoError(t, err)
	assert.Equal(t, "old-refresh", token.RefreshToken)
}

func TestBuildOAuthAuth_ReturnsExplicitError(t *testing.T) {
	// BuildOAuthAuth returns an explicit error until automatic server metadata
	// discovery is implemented. The device flow itself (RunDeviceFlow) works;
	// BuildOAuthAuth is the discovery-layer gateway.
	err := BuildOAuthAuth("test-server", "http://localhost:8080")
	require.Error(t, err, "BuildOAuthAuth must return an error — silent false is dishonest")
	assert.Contains(t, err.Error(), "not yet implemented")
	assert.Contains(t, err.Error(), "test-server")
}

// TestRunDeviceFlow_SuccessOnFirstPoll verifies that RunDeviceFlow returns a
// valid OAuthToken when the token endpoint grants on the first poll.
func TestRunDeviceFlow_SuccessOnFirstPoll(t *testing.T) {
	// Mock token endpoint: first call returns authorization_pending, second grants.
	pollCount := 0
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pollCount++
		if pollCount == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "authorization_pending",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "granted-token",
			"refresh_token": "refresh-tok",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "read",
		})
	}))
	defer tokenSrv.Close()

	// Mock device authorization endpoint.
	deviceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_code":      "dev-code-123",
			"user_code":        "ABCD-1234",
			"verification_uri": "https://example.com/activate",
			"expires_in":       300,
			"interval":         1,
		})
	}))
	defer deviceSrv.Close()

	cfg := DeviceFlowConfig{
		DeviceAuthURL: deviceSrv.URL,
		TokenURL:      tokenSrv.URL,
		ClientID:      "test-client",
		Scopes:        "read",
		PollInterval:  100 * time.Millisecond,
		Timeout:       10 * time.Second,
	}

	token, err := RunDeviceFlow(cfg)
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "granted-token", token.AccessToken)
	assert.Equal(t, "refresh-tok", token.RefreshToken)
	assert.False(t, token.ExpiresAt.IsZero())
	assert.Equal(t, 2, pollCount)
}

// TestRunDeviceFlow_AccessDenied verifies that RunDeviceFlow returns an error
// when the user denies authorization.
func TestRunDeviceFlow_AccessDenied(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             "access_denied",
			"error_description": "user denied",
		})
	}))
	defer tokenSrv.Close()

	deviceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_code": "dev-code", "user_code": "CODE", "verification_uri": "http://x.com",
			"expires_in": 300, "interval": 1,
		})
	}))
	defer deviceSrv.Close()

	_, err := RunDeviceFlow(DeviceFlowConfig{
		DeviceAuthURL: deviceSrv.URL,
		TokenURL:      tokenSrv.URL,
		ClientID:      "client",
		PollInterval:  50 * time.Millisecond,
		Timeout:       5 * time.Second,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "denied")
}

// TestRunDeviceFlow_Timeout verifies that RunDeviceFlow returns a timeout error.
func TestRunDeviceFlow_Timeout(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "authorization_pending"})
	}))
	defer tokenSrv.Close()

	deviceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_code": "dev-code", "user_code": "CODE", "verification_uri": "http://x.com",
			"expires_in": 300, "interval": 1,
		})
	}))
	defer deviceSrv.Close()

	_, err := RunDeviceFlow(DeviceFlowConfig{
		DeviceAuthURL: deviceSrv.URL,
		TokenURL:      tokenSrv.URL,
		ClientID:      "client",
		PollInterval:  50 * time.Millisecond,
		Timeout:       150 * time.Millisecond,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestRemoveOAuthTokens(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)

	store := NewOAuthTokenStore(dir)
	store.Set("test-server", &OAuthToken{AccessToken: "tok"})

	// Verify token exists
	tokensFile := filepath.Join(dir, "mcp_oauth_tokens.json")
	data, err := os.ReadFile(tokensFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "tok")

	RemoveOAuthTokens("test-server")

	// Reload and verify removed
	store2 := NewOAuthTokenStore(dir)
	assert.Nil(t, store2.Get("test-server"))
}
