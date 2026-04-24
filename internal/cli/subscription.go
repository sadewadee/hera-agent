package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const nousPortalAPI = "https://portal.nousresearch.com/api/v1"

// SubscriptionInfo holds Nous Portal subscription details.
type SubscriptionInfo struct {
	Plan            string    `json:"plan"`
	TokensRemaining int       `json:"tokens_remaining"`
	TokensUsed      int       `json:"tokens_used"`
	Expiry          time.Time `json:"expiry"`
	Active          bool      `json:"active"`
}

// CheckSubscription calls the Nous Portal API to retrieve subscription status.
func CheckSubscription(apiKey string) (*SubscriptionInfo, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("no API key provided; set NOUS_API_KEY or configure providers.nous.api_key")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, nousPortalAPI+"/subscription", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("subscription request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var info SubscriptionInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &info, nil
}

// FormatSubscription returns a human-readable subscription summary.
func FormatSubscription(info *SubscriptionInfo) string {
	if info == nil {
		return "No subscription information available."
	}

	status := "inactive"
	if info.Active {
		status = "active"
	}

	return fmt.Sprintf("Nous Portal Subscription\n"+
		"  Plan:            %s\n"+
		"  Status:          %s\n"+
		"  Tokens remaining: %d\n"+
		"  Tokens used:     %d\n"+
		"  Expires:         %s",
		info.Plan,
		status,
		info.TokensRemaining,
		info.TokensUsed,
		info.Expiry.Format("2006-01-02"),
	)
}
