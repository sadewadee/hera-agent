package hcore

// Version is the single source of truth for the Hera semantic version.
// Referenced by health endpoints, tool client-version headers, and CLI tests.
const (
	Version        = "0.0.142"
	AppName        = "hera"
	DefaultPort    = 8080
	MaxMessageSize = 100 * 1024
	MaxFileSize    = 10 * 1024 * 1024
	DefaultDBName  = "hera.db"
	ConfigFileName = "hera.yaml"
	SOULFileName   = "SOUL.md"
	HintFileName   = ".hera.md"
	DefaultModel   = "gpt-4o"
	DefaultTimeout = 120
)

var DefaultSkillDirs = []string{"skills", "optional-skills"}
var SupportedPlatforms = []string{"cli", "telegram", "discord", "slack", "whatsapp", "signal", "matrix", "email", "sms", "homeassistant", "dingtalk", "feishu", "wecom", "mattermost", "bluebubbles", "webhook", "apiserver", "mcp"}
