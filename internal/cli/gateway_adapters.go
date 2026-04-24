package cli

import (
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/gateway/platforms"
)

// registerPlatformAdapters registers all enabled platform adapters with the gateway
// based on the configuration. Returns the number of adapters registered.
func registerPlatformAdapters(gw *gateway.Gateway, cfg *config.Config) int {
	count := 0

	for name, platCfg := range cfg.Gateway.Platforms {
		if !platCfg.Enabled {
			continue
		}

		var adapter gateway.PlatformAdapter

		switch name {
		case "cli":
			adapter = platforms.NewCLIAdapter()
		case "telegram":
			if platCfg.Token == "" {
				continue
			}
			if platCfg.Extra["mode"] == "webhook" {
				adapter = platforms.NewTelegramAdapterWithOptions(platCfg.Token, platforms.TelegramOptions{
					Mode:        "webhook",
					WebhookURL:  platCfg.Extra["webhook_url"],
					WebhookAddr: platCfg.Extra["webhook_addr"],
					WebhookPath: platCfg.Extra["webhook_path"],
				})
			} else {
				adapter = platforms.NewTelegramAdapter(platCfg.Token)
			}
		case "discord":
			if platCfg.Token == "" {
				continue
			}
			adapter = platforms.NewDiscordAdapter(platCfg.Token)
		case "slack":
			botToken := platCfg.Token
			appToken := platCfg.Extra["app_token"]
			if botToken == "" || appToken == "" {
				continue
			}
			adapter = platforms.NewSlackAdapter(botToken, appToken)
		case "apiserver":
			addr := platCfg.Extra["addr"]
			adapter = platforms.NewAPIServerAdapter(addr)
		case "webhook":
			adapter = platforms.NewWebhookAdapter(platforms.WebhookConfig{
				Addr:        platCfg.Extra["addr"],
				Secret:      platCfg.Extra["secret"],
				CallbackURL: platCfg.Extra["callback_url"],
			})
		case "threads":
			if platCfg.Token == "" {
				continue
			}
			adapter = platforms.NewThreadsAdapter(platCfg.Token)
		case "whatsapp":
			if platCfg.Token == "" {
				continue
			}
			adapter = platforms.NewWhatsAppAdapter(platforms.WhatsAppConfig{
				PhoneNumberID: platCfg.Extra["phone_number_id"],
				AccessToken:   platCfg.Token,
				VerifyToken:   platCfg.Extra["verify_token"],
				CallbackAddr:  platCfg.Extra["callback_addr"],
			})
		case "signal":
			if platCfg.Extra["api_url"] == "" {
				continue
			}
			adapter = platforms.NewSignalAdapter(platforms.SignalConfig{
				APIURL:      platCfg.Extra["api_url"],
				PhoneNumber: platCfg.Extra["phone_number"],
			})
		case "matrix":
			if platCfg.Token == "" {
				continue
			}
			adapter = platforms.NewMatrixAdapter(platforms.MatrixConfig{
				HomeserverURL: platCfg.Extra["homeserver_url"],
				UserID:        platCfg.Extra["user_id"],
				AccessToken:   platCfg.Token,
			})
		case "email":
			if platCfg.Extra["smtp_host"] == "" {
				continue
			}
			adapter = platforms.NewEmailAdapter(platforms.EmailConfig{
				SMTPHost:     platCfg.Extra["smtp_host"],
				SMTPPort:     platCfg.Extra["smtp_port"],
				SMTPUser:     platCfg.Extra["smtp_user"],
				SMTPPassword: platCfg.Extra["smtp_password"],
				FromAddress:  platCfg.Extra["from_address"],
				WebhookAddr:  platCfg.Extra["webhook_addr"],
			})
		case "sms":
			if platCfg.Extra["account_sid"] == "" {
				continue
			}
			adapter = platforms.NewSMSAdapter(platforms.SMSConfig{
				AccountSID:  platCfg.Extra["account_sid"],
				AuthToken:   platCfg.Token,
				FromNumber:  platCfg.Extra["from_number"],
				WebhookAddr: platCfg.Extra["webhook_addr"],
			})
		case "homeassistant":
			if platCfg.Token == "" {
				continue
			}
			adapter = platforms.NewHomeAssistantAdapter(platforms.HomeAssistantConfig{
				HAURL:       platCfg.Extra["ha_url"],
				Token:       platCfg.Token,
				WebhookAddr: platCfg.Extra["webhook_addr"],
			})
		case "dingtalk":
			if platCfg.Token == "" {
				continue
			}
			adapter = platforms.NewDingTalkAdapter(platforms.DingTalkConfig{
				AccessToken:  platCfg.Token,
				Secret:       platCfg.Extra["secret"],
				CallbackAddr: platCfg.Extra["callback_addr"],
			})
		case "feishu":
			if platCfg.Extra["app_id"] == "" {
				continue
			}
			adapter = platforms.NewFeishuAdapter(platforms.FeishuConfig{
				AppID:             platCfg.Extra["app_id"],
				AppSecret:         platCfg.Extra["app_secret"],
				VerificationToken: platCfg.Extra["verification_token"],
				CallbackAddr:      platCfg.Extra["callback_addr"],
			})
		case "wecom":
			if platCfg.Extra["corp_id"] == "" {
				continue
			}
			adapter = platforms.NewWeComAdapter(platforms.WeComConfig{
				CorpID:       platCfg.Extra["corp_id"],
				CorpSecret:   platCfg.Extra["corp_secret"],
				VerifyToken:  platCfg.Extra["verify_token"],
				CallbackAddr: platCfg.Extra["callback_addr"],
			})
		case "mattermost":
			if platCfg.Token == "" {
				continue
			}
			adapter = platforms.NewMattermostAdapter(platforms.MattermostConfig{
				ServerURL: platCfg.Extra["server_url"],
				Token:     platCfg.Token,
			})
		case "bluebubbles":
			if platCfg.Extra["api_url"] == "" {
				continue
			}
			adapter = platforms.NewBlueBubblesAdapter(platforms.BlueBubblesConfig{
				APIURL:   platCfg.Extra["api_url"],
				Password: platCfg.Extra["password"],
			})
		default:
			// Unknown platform -- skip.
			continue
		}

		if adapter != nil {
			gw.AddAdapter(adapter)
			count++
		}
	}

	return count
}
