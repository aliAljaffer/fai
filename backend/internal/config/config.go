package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL          string
	Port                 string
	BindAddr             string
	DisableAuth          bool
	OIDCIssuerURL        string
	OIDCAudience         string
	JiraBaseURL          string
	JiraEmail            string
	JiraAPIToken         string
	JiraDefaultProject   string
	JiraStatusColumnMap  string
	JiraEpicIssueType    string
	JiraTaskIssueType    string
	JiraSubtaskIssueType string
	JiraEpicLinkField    string
	NtfyURL              string
	NtfyTopic            string
	NtfyUsername         string
	NtfyPassword         string
	SMTPAddr             string
	SMTPFrom             string
	SMTPTo               string
	ICSToken             string
	PublicURL            string
	AttachmentsDir       string
	AIProvider           string
	AIBaseURL            string
	AIAPIKey             string
	AIModel              string
	// TelegramBotToken empty disables the assistant bot.
	TelegramBotToken      string
	TelegramAllowedUserID int64
	TelegramAllowedChatID int64
	TelegramStateFile     string
	// ReminderChannel selects one delivery channel when set. Empty keeps the
	// legacy behavior of sending to every configured channel.
	ReminderChannel string
	// StaticDir serves the built frontend SPA from the same origin as the API.
	// Used by the Electron build, which has no separate web server.
	StaticDir string
	// AuthMode is "oidc" (default: backend validates bearer JWTs itself, used by the
	// kubernetes deployment) or "proxy-header" (trusts X-Forwarded-User/-Email set by
	// a fronting oauth2-proxy, used by the docker-compose deployment). Only safe in
	// "proxy-header" mode if the backend is unreachable except through that proxy.
	AuthMode string
}

// IsSQLite reports whether DatabaseURL names a SQLite file rather than a Postgres
// connection string, selecting which store.Store implementation main.go constructs.
func (c Config) IsSQLite() bool {
	return strings.HasPrefix(c.DatabaseURL, "sqlite://") || strings.HasPrefix(c.DatabaseURL, "sqlite:")
}

// SQLitePath strips the sqlite:// (or sqlite:) scheme prefix to get the bare file path.
func (c Config) SQLitePath() string {
	s := strings.TrimPrefix(c.DatabaseURL, "sqlite://")
	return strings.TrimPrefix(s, "sqlite:")
}

func Load() Config {
	return Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		Port:                  envOr("PORT", "8080"),
		BindAddr:              os.Getenv("BIND_ADDR"),
		DisableAuth:           os.Getenv("DISABLE_AUTH") == "true",
		OIDCIssuerURL:         os.Getenv("OIDC_ISSUER_URL"),
		OIDCAudience:          os.Getenv("OIDC_AUDIENCE"),
		JiraBaseURL:           os.Getenv("JIRA_BASE_URL"),
		JiraEmail:             os.Getenv("JIRA_EMAIL"),
		JiraAPIToken:          os.Getenv("JIRA_API_TOKEN"),
		JiraDefaultProject:    os.Getenv("JIRA_DEFAULT_PROJECT_KEY"),
		JiraStatusColumnMap:   os.Getenv("JIRA_STATUS_COLUMN_MAP"),
		JiraEpicIssueType:     envOr("JIRA_EPIC_ISSUE_TYPE", "Epic"),
		JiraTaskIssueType:     envOr("JIRA_TASK_ISSUE_TYPE", "Task"),
		JiraSubtaskIssueType:  envOr("JIRA_SUBTASK_ISSUE_TYPE", "Sub-task"),
		JiraEpicLinkField:     envOr("JIRA_EPIC_LINK_FIELD", "parent"),
		NtfyURL:               os.Getenv("NTFY_URL"),
		NtfyTopic:             os.Getenv("NTFY_TOPIC"),
		NtfyUsername:          os.Getenv("NTFY_USERNAME"),
		NtfyPassword:          os.Getenv("NTFY_PASSWORD"),
		SMTPAddr:              os.Getenv("SMTP_ADDR"),
		SMTPFrom:              os.Getenv("SMTP_FROM"),
		SMTPTo:                os.Getenv("SMTP_TO"),
		ICSToken:              os.Getenv("ICS_TOKEN"),
		PublicURL:             os.Getenv("PUBLIC_URL"),
		AttachmentsDir:        os.Getenv("ATTACHMENTS_DIR"),
		AIProvider:            os.Getenv("AI_PROVIDER"),
		AIBaseURL:             os.Getenv("AI_BASE_URL"),
		AIAPIKey:              os.Getenv("AI_API_KEY"),
		AIModel:               os.Getenv("AI_MODEL"),
		TelegramBotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramAllowedUserID: envInt64("TELEGRAM_ALLOWED_USER_ID"),
		TelegramStateFile:     envOr("TELEGRAM_STATE_FILE", "fai-telegram-offset"),
		TelegramAllowedChatID: envInt64("TELEGRAM_ALLOWED_CHAT_ID"),
		ReminderChannel:       strings.ToLower(strings.TrimSpace(os.Getenv("REMINDER_CHANNEL"))),
		StaticDir:             os.Getenv("STATIC_DIR"),
		AuthMode:              envOr("AUTH_MODE", "oidc"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt64(key string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(key)), 10, 64)
	if err != nil {
		return 0
	}
	return value
}
