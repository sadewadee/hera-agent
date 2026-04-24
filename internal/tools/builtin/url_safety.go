package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// safeBrowsingEndpoint is a package-level variable so tests can swap it for
// an httptest server without modifying production code.
var safeBrowsingEndpoint = "https://safebrowsing.googleapis.com/v4/threatMatches:find"

// URLSafetyTool checks URL safety using the Google Safe Browsing API v4.
// Requires GOOGLE_SAFE_BROWSING_KEY environment variable.
// Returns SAFE when the API reports no threats, UNSAFE with threat types
// when matches are found, and an error when the API key is not configured.
//
// apiKey and client are resolved at construction time by NewURLSafetyTool
// so Execute is safe for concurrent use without locking.
type URLSafetyTool struct {
	apiKey        string
	client        *http.Client
	clientVersion string
}

// NewURLSafetyTool reads GOOGLE_SAFE_BROWSING_KEY once at construction
// and returns a tool with an initialised HTTP client. The clientVersion
// is used for Google's quota attribution and mirrors the binary version.
func NewURLSafetyTool(clientVersion string) *URLSafetyTool {
	return &URLSafetyTool{
		apiKey:        os.Getenv("GOOGLE_SAFE_BROWSING_KEY"),
		client:        &http.Client{Timeout: 10 * time.Second},
		clientVersion: clientVersion,
	}
}

type urlSafetyArgs struct {
	URL string `json:"url"`
}

func (t *URLSafetyTool) Name() string { return "url_safety" }
func (t *URLSafetyTool) Description() string {
	return "Checks URL safety using Google Safe Browsing API v4. Requires GOOGLE_SAFE_BROWSING_KEY."
}
func (t *URLSafetyTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"URL to check for safety"}},"required":["url"]}`)
}

func (t *URLSafetyTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a urlSafetyArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	if a.URL == "" {
		return &tools.Result{Content: "url is required", IsError: true}, nil
	}

	if t.apiKey == "" {
		return &tools.Result{
			Content: "url_safety: GOOGLE_SAFE_BROWSING_KEY not set; cannot check URL safety",
			IsError: true,
		}, nil
	}

	clientVersion := t.clientVersion
	if clientVersion == "" {
		clientVersion = "0.0.0"
	}
	// Build Safe Browsing v4 request body.
	reqBody := map[string]interface{}{
		"client": map[string]string{
			"clientId":      "hera",
			"clientVersion": clientVersion,
		},
		"threatInfo": map[string]interface{}{
			"threatTypes":      []string{"MALWARE", "SOCIAL_ENGINEERING", "UNWANTED_SOFTWARE", "POTENTIALLY_HARMFUL_APPLICATION"},
			"platformTypes":    []string{"ANY_PLATFORM"},
			"threatEntryTypes": []string{"URL"},
			"threatEntries":    []map[string]string{{"url": a.URL}},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("url_safety: build request: %v", err), IsError: true}, nil
	}

	endpoint := fmt.Sprintf("%s?key=%s", safeBrowsingEndpoint, t.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("url_safety: create request: %v", err), IsError: true}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("url_safety: request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &tools.Result{
			Content: fmt.Sprintf("url_safety: API returned HTTP %d", resp.StatusCode),
			IsError: true,
		}, nil
	}

	var result struct {
		Matches []struct {
			ThreatType string `json:"threatType"`
		} `json:"matches"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &tools.Result{Content: fmt.Sprintf("url_safety: decode response: %v", err), IsError: true}, nil
	}

	if len(result.Matches) == 0 {
		return &tools.Result{Content: fmt.Sprintf("SAFE: %s — no threats detected by Google Safe Browsing", a.URL)}, nil
	}

	threats := make([]string, 0, len(result.Matches))
	for _, m := range result.Matches {
		threats = append(threats, m.ThreatType)
	}
	return &tools.Result{
		Content: fmt.Sprintf("UNSAFE: %s — threat types detected: %v", a.URL, threats),
		IsError: true,
	}, nil
}

// RegisterURLSafety registers the url_safety tool with the version string
// used for Google Safe Browsing quota attribution.
func RegisterURLSafety(registry *tools.Registry, clientVersion string) {
	registry.Register(NewURLSafetyTool(clientVersion))
}
