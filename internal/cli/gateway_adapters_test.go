package cli

import (
	"testing"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/gateway"
)

// TestRegisterPlatformAdapters verifies that every named platform produces a
// non-nil adapter (i.e. its constructor is reached) when minimal required
// credentials are present in the config.
func TestRegisterPlatformAdapters(t *testing.T) {
	cases := []struct {
		platform string
		cfg      config.PlatformConfig
	}{
		{
			"cli",
			config.PlatformConfig{Enabled: true},
		},
		{
			"telegram",
			config.PlatformConfig{Enabled: true, Token: "tok"},
		},
		{
			"discord",
			config.PlatformConfig{Enabled: true, Token: "tok"},
		},
		{
			"slack",
			config.PlatformConfig{Enabled: true, Token: "bot-tok", Extra: map[string]string{"app_token": "app-tok"}},
		},
		{
			"apiserver",
			config.PlatformConfig{Enabled: true},
		},
		{
			"webhook",
			config.PlatformConfig{Enabled: true},
		},
		{
			"threads",
			config.PlatformConfig{Enabled: true, Token: "tok"},
		},
		{
			"whatsapp",
			config.PlatformConfig{Enabled: true, Token: "tok", Extra: map[string]string{"phone_number_id": "111"}},
		},
		{
			"signal",
			config.PlatformConfig{Enabled: true, Extra: map[string]string{"api_url": "http://localhost:8080", "phone_number": "+1"}},
		},
		{
			"matrix",
			config.PlatformConfig{Enabled: true, Token: "tok", Extra: map[string]string{"homeserver_url": "https://matrix.org", "user_id": "@bot:matrix.org"}},
		},
		{
			"email",
			config.PlatformConfig{Enabled: true, Extra: map[string]string{"smtp_host": "smtp.example.com", "smtp_port": "587", "smtp_user": "u", "smtp_password": "p", "from_address": "bot@example.com"}},
		},
		{
			"sms",
			config.PlatformConfig{Enabled: true, Token: "auth-tok", Extra: map[string]string{"account_sid": "ACxxx", "from_number": "+1555"}},
		},
		{
			"homeassistant",
			config.PlatformConfig{Enabled: true, Token: "tok", Extra: map[string]string{"ha_url": "http://homeassistant.local:8123"}},
		},
		{
			"dingtalk",
			config.PlatformConfig{Enabled: true, Token: "tok"},
		},
		{
			"feishu",
			config.PlatformConfig{Enabled: true, Extra: map[string]string{"app_id": "appid", "app_secret": "secret"}},
		},
		{
			"wecom",
			config.PlatformConfig{Enabled: true, Extra: map[string]string{"corp_id": "corpid", "corp_secret": "secret"}},
		},
		{
			"mattermost",
			config.PlatformConfig{Enabled: true, Token: "tok", Extra: map[string]string{"server_url": "http://mm.example.com"}},
		},
		{
			"bluebubbles",
			config.PlatformConfig{Enabled: true, Extra: map[string]string{"api_url": "http://localhost:1234", "password": "pass"}},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.platform, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Gateway.Platforms = map[string]config.PlatformConfig{
				tc.platform: tc.cfg,
			}
			gw := gateway.NewGateway(gateway.GatewayOptions{})
			count := registerPlatformAdapters(gw, cfg)
			if count != 1 {
				t.Errorf("platform %q: registerPlatformAdapters returned count=%d, want 1", tc.platform, count)
			}
		})
	}
}

// TestRegisterPlatformAdapters_MissingCredentials verifies that platforms
// requiring credentials are silently skipped (count=0) when tokens are absent.
func TestRegisterPlatformAdapters_MissingCredentials(t *testing.T) {
	cases := []struct {
		platform string
		cfg      config.PlatformConfig
	}{
		{"telegram", config.PlatformConfig{Enabled: true}},
		{"discord", config.PlatformConfig{Enabled: true}},
		{"slack", config.PlatformConfig{Enabled: true, Token: "bot"}}, // missing app_token
		{"threads", config.PlatformConfig{Enabled: true}},
		{"whatsapp", config.PlatformConfig{Enabled: true}},      // missing token
		{"signal", config.PlatformConfig{Enabled: true}},        // missing api_url
		{"matrix", config.PlatformConfig{Enabled: true}},        // missing token
		{"email", config.PlatformConfig{Enabled: true}},         // missing smtp_host
		{"sms", config.PlatformConfig{Enabled: true}},           // missing account_sid
		{"homeassistant", config.PlatformConfig{Enabled: true}}, // missing token
		{"dingtalk", config.PlatformConfig{Enabled: true}},
		{"feishu", config.PlatformConfig{Enabled: true}}, // missing app_id
		{"wecom", config.PlatformConfig{Enabled: true}},  // missing corp_id
		{"mattermost", config.PlatformConfig{Enabled: true}},
		{"bluebubbles", config.PlatformConfig{Enabled: true}}, // missing api_url
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.platform+"_missing_creds", func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Gateway.Platforms = map[string]config.PlatformConfig{
				tc.platform: tc.cfg,
			}
			gw := gateway.NewGateway(gateway.GatewayOptions{})
			count := registerPlatformAdapters(gw, cfg)
			if count != 0 {
				t.Errorf("platform %q with missing creds: expected count=0, got %d", tc.platform, count)
			}
		})
	}
}
