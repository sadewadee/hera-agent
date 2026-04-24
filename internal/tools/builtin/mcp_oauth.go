// Package builtin provides built-in tool implementations.
//
// mcp_oauth.go implements OAuth 2.1 PKCE authentication flow for MCP
// servers that require it. Handles token acquisition, storage, refresh,
// and cleanup.
package builtin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/paths"
)

// OAuthTokenStore manages OAuth tokens for MCP servers.
type OAuthTokenStore struct {
	mu     sync.Mutex
	tokens map[string]*OAuthToken
	path   string
}

// OAuthToken holds OAuth access and refresh tokens.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

// NewOAuthTokenStore creates a store backed by the given file path.
func NewOAuthTokenStore(heraHome string) *OAuthTokenStore {
	store := &OAuthTokenStore{
		tokens: make(map[string]*OAuthToken),
		path:   filepath.Join(heraHome, "mcp_oauth_tokens.json"),
	}
	store.load()
	return store
}

func (s *OAuthTokenStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &s.tokens)
}

func (s *OAuthTokenStore) save() {
	data, err := json.MarshalIndent(s.tokens, "", "  ")
	if err != nil {
		slog.Warn("failed to save OAuth tokens", "error", err)
		return
	}
	dir := filepath.Dir(s.path)
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(s.path, data, 0o600)
}

// Get returns the token for a server, or nil if not found/expired.
func (s *OAuthTokenStore) Get(serverName string) *OAuthToken {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[serverName]
	if !ok {
		return nil
	}
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		return nil
	}
	return token
}

// Set stores a token for a server.
func (s *OAuthTokenStore) Set(serverName string, token *OAuthToken) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[serverName] = token
	s.save()
}

// Remove deletes tokens for a server.
func (s *OAuthTokenStore) Remove(serverName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, serverName)
	s.save()
}

// DeviceFlowConfig holds the parameters needed to run an OAuth 2.0 device flow
// (RFC 8628) for an MCP server. All fields are required.
type DeviceFlowConfig struct {
	// DeviceAuthURL is the device authorization endpoint
	// (e.g. "https://auth.example.com/oauth2/device/authorize").
	DeviceAuthURL string
	// TokenURL is the token endpoint
	// (e.g. "https://auth.example.com/oauth2/token").
	TokenURL string
	// ClientID is the OAuth2 client identifier.
	ClientID string
	// Scopes is the space-separated list of scopes to request.
	Scopes string
	// PollInterval is how long to wait between token poll attempts.
	// Defaults to 5 seconds if zero.
	PollInterval time.Duration
	// Timeout is the total time to wait for the user to authorize.
	// Defaults to 5 minutes if zero.
	Timeout time.Duration
}

// deviceAuthResponse is the JSON response from the device authorization endpoint.
type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// deviceTokenResponse is the JSON response from the token endpoint during polling.
type deviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// RunDeviceFlow executes the OAuth 2.0 device authorization flow (RFC 8628).
// It requests a device code, prints the user code and verification URL to
// stderr, then polls the token endpoint until the user authorizes or the
// device code expires. Returns the granted OAuthToken on success.
func RunDeviceFlow(cfg DeviceFlowConfig) (*OAuthToken, error) {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Minute
	}

	// Step 1: request device + user codes.
	form := url.Values{
		"client_id": {cfg.ClientID},
	}
	if cfg.Scopes != "" {
		form.Set("scope", cfg.Scopes)
	}
	resp, err := http.PostForm(cfg.DeviceAuthURL, form)
	if err != nil {
		return nil, fmt.Errorf("device auth request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device auth response: HTTP %d", resp.StatusCode)
	}

	var dar deviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&dar); err != nil {
		return nil, fmt.Errorf("decode device auth response: %w", err)
	}

	// Use server-advertised interval if larger than our default.
	if dar.Interval > 0 {
		serverInterval := time.Duration(dar.Interval) * time.Second
		if serverInterval > cfg.PollInterval {
			cfg.PollInterval = serverInterval
		}
	}

	// Print instructions for the user.
	verifyURL := dar.VerificationURI
	if dar.VerificationURIComplete != "" {
		verifyURL = dar.VerificationURIComplete
	}
	slog.Info("OAuth device flow: please authorize",
		"user_code", dar.UserCode,
		"url", verifyURL,
	)
	fmt.Printf("\nTo authorize hera-mcp, visit:\n  %s\nand enter code: %s\n\n", verifyURL, dar.UserCode)

	// Step 2: poll until authorized, expired, or timed out.
	deadline := time.Now().Add(cfg.Timeout)
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C

		pollForm := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {dar.DeviceCode},
			"client_id":   {cfg.ClientID},
		}
		pollResp, pollErr := http.PostForm(cfg.TokenURL, pollForm)
		if pollErr != nil {
			slog.Debug("device flow poll error (will retry)", "error", pollErr)
			continue
		}

		var dtr deviceTokenResponse
		_ = json.NewDecoder(pollResp.Body).Decode(&dtr)
		pollResp.Body.Close()

		switch dtr.Error {
		case "":
			// Success.
			if dtr.AccessToken == "" {
				continue
			}
			token := &OAuthToken{
				AccessToken:  dtr.AccessToken,
				RefreshToken: dtr.RefreshToken,
				TokenType:    dtr.TokenType,
				Scope:        dtr.Scope,
			}
			if dtr.ExpiresIn > 0 {
				token.ExpiresAt = time.Now().Add(time.Duration(dtr.ExpiresIn) * time.Second)
			}
			return token, nil
		case "authorization_pending":
			// Normal — user hasn't approved yet.
			continue
		case "slow_down":
			// Server asked us to back off.
			cfg.PollInterval += 5 * time.Second
			ticker.Reset(cfg.PollInterval)
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired; restart the authorization flow")
		case "access_denied":
			return nil, fmt.Errorf("user denied authorization")
		default:
			return nil, fmt.Errorf("device flow error %q: %s", dtr.Error, dtr.ErrorDesc)
		}
	}

	return nil, fmt.Errorf("device flow timed out after %s", cfg.Timeout)
}

// BuildOAuthAuth initiates an OAuth 2.0 device flow for an MCP server.
// serverName is used as a key in the OAuthTokenStore.
// serverURL is reserved for future server-metadata discovery (RFC 8414);
// callers currently provide DeviceFlowConfig directly via RunDeviceFlow.
//
// If a valid cached token exists for serverName it is returned immediately
// without starting a new flow.
func BuildOAuthAuth(serverName, serverURL string) error {
	slog.Info("OAuth device flow requested for MCP server",
		"server", serverName,
		"url", serverURL,
	)
	// serverURL is reserved for RFC 8414 server metadata discovery which
	// would auto-detect the device_authorization_endpoint and token_endpoint.
	// Until that discovery layer is implemented, callers must provide
	// DeviceFlowConfig.DeviceAuthURL and TokenURL explicitly via RunDeviceFlow.
	// This function is the gateway that will wire discovery when ready.
	return fmt.Errorf("BuildOAuthAuth: automatic server metadata discovery not yet implemented; "+
		"use RunDeviceFlow(DeviceFlowConfig{...}) directly for server %q — see ROADMAP.md for v0.12.2 timeline", serverName)
}

// RemoveOAuthTokens cleans up OAuth tokens for a removed MCP server.
func RemoveOAuthTokens(serverName string) {
	store := NewOAuthTokenStore(paths.HeraHome())
	store.Remove(serverName)
}

// PKCE helpers.

// GenerateCodeVerifier creates a random code verifier for PKCE.
func GenerateCodeVerifier() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// GenerateCodeChallenge creates an S256 code challenge from a verifier.
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ExchangeAuthCode exchanges an authorization code for tokens.
func ExchangeAuthCode(tokenURL, code, redirectURI, clientID, codeVerifier string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"code_verifier": {codeVerifier},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	token := &OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}
	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	return token, nil
}

// RefreshAccessToken uses a refresh token to get a new access token.
func RefreshAccessToken(tokenURL, refreshToken, clientID string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	token := &OAuthToken{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
	}
	if tokenResp.RefreshToken != "" {
		token.RefreshToken = tokenResp.RefreshToken
	} else {
		token.RefreshToken = refreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	return token, nil
}

func init() {
	_ = strings.TrimSpace // keep strings import for future metadata discovery
}
