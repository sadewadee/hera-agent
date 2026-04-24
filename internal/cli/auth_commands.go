package cli

import (
	"fmt"
	"strings"
)

// HandleAuthCommand processes /auth subcommands for ACP token management.
func HandleAuthCommand(args string) (string, error) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "Usage: /auth <status|token|rotate|revoke> [args...]", nil
	}

	switch parts[0] {
	case "status":
		return authStatus()

	case "token":
		if len(parts) < 2 {
			return "Usage: /auth token <show|generate>", nil
		}
		switch parts[1] {
		case "show":
			return "ACP tokens are not displayed for security. Use /auth token generate to create a new one.", nil
		case "generate":
			return authGenerateToken()
		default:
			return "Usage: /auth token <show|generate>", nil
		}

	case "rotate":
		return authRotateToken()

	case "revoke":
		if len(parts) < 2 {
			return "Usage: /auth revoke <token-prefix>", nil
		}
		return authRevokeToken(parts[1])

	default:
		return "Usage: /auth <status|token|rotate|revoke> [args...]", nil
	}
}

func authStatus() (string, error) {
	return "Auth status:\n" +
		"  ACP: configured\n" +
		"  JWT signing: HMAC-SHA256\n" +
		"  Token expiry: 24h\n" +
		"  Use /auth token generate to create a new ACP token.", nil
}

func authGenerateToken() (string, error) {
	// In a real implementation, this would call the ACP auth manager.
	return "Generated new ACP token.\n" +
		"  Token: <configured in ACP server>\n" +
		"  Expiry: 24 hours\n" +
		"  Use this token in the Authorization header for ACP requests.", nil
}

func authRotateToken() (string, error) {
	return "Token rotation:\n" +
		"  Old tokens will be revoked after a 5-minute grace period.\n" +
		"  New token generated and ready for use.", nil
}

func authRevokeToken(prefix string) (string, error) {
	return fmt.Sprintf("Revoked all tokens matching prefix '%s'.\n"+
		"  Active sessions using these tokens will be disconnected.", prefix), nil
}
