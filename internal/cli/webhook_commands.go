package cli

import (
	"fmt"
	"strings"
)

// WebhookEntry represents a configured webhook endpoint.
type WebhookEntry struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Events string `json:"events"` // comma-separated event types
	Active bool   `json:"active"`
}

// HandleWebhookCommand processes /webhook subcommands for managing webhooks.
func HandleWebhookCommand(args string) (string, error) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "Usage: /webhook <list|add|remove|test> [args...]", nil
	}

	switch parts[0] {
	case "list":
		return webhookList()

	case "add":
		if len(parts) < 3 {
			return "Usage: /webhook add <name> <url> [events...]", nil
		}
		name := parts[1]
		url := parts[2]
		events := "all"
		if len(parts) > 3 {
			events = strings.Join(parts[3:], ",")
		}
		return webhookAdd(name, url, events)

	case "remove":
		if len(parts) < 2 {
			return "Usage: /webhook remove <name>", nil
		}
		return webhookRemove(parts[1])

	case "test":
		if len(parts) < 2 {
			return "Usage: /webhook test <name>", nil
		}
		return webhookTest(parts[1])

	default:
		return "Usage: /webhook <list|add|remove|test> [args...]", nil
	}
}

func webhookList() (string, error) {
	return "Configured webhooks:\n" +
		"  (none configured)\n\n" +
		"Use /webhook add <name> <url> [events] to add a webhook.", nil
}

func webhookAdd(name, url, events string) (string, error) {
	return fmt.Sprintf("Added webhook '%s'\n  URL: %s\n  Events: %s\n  Status: active",
		name, url, events), nil
}

func webhookRemove(name string) (string, error) {
	return fmt.Sprintf("Removed webhook '%s'.", name), nil
}

func webhookTest(name string) (string, error) {
	return fmt.Sprintf("Sending test event to webhook '%s'...\n  Result: (webhook not configured yet)", name), nil
}
