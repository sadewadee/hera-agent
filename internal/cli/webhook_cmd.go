package cli

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WebhookTestCommand sends a test webhook payload.
func WebhookTestCommand(url string, payload string) string {
	if url == "" { return "Error: webhook URL required" }
	if payload == "" { payload = `{"event":"test","timestamp":"` + time.Now().Format(time.RFC3339) + `"}` }
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(payload))
	if err != nil { return fmt.Sprintf("Webhook test failed: %v", err) }
	defer resp.Body.Close()
	return fmt.Sprintf("Webhook test sent to %s - Status: %d", url, resp.StatusCode)
}
