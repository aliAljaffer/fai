package config

import "os"

type Config struct {
	DatabaseURL         string
	Port                string
	DisableAuth         bool
	OIDCIssuerURL       string
	OIDCAudience        string
	JiraBaseURL         string
	JiraEmail           string
	JiraAPIToken        string
	JiraDefaultProject  string
	JiraStatusColumnMap string
	NtfyURL             string
	NtfyTopic           string
	NtfyUsername        string
	NtfyPassword        string
	AIProvider          string
	AIAPIKey            string
	AIModel             string
}

func Load() Config {
	return Config{
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		Port:                envOr("PORT", "8080"),
		DisableAuth:         os.Getenv("DISABLE_AUTH") == "true",
		OIDCIssuerURL:       os.Getenv("OIDC_ISSUER_URL"),
		OIDCAudience:        os.Getenv("OIDC_AUDIENCE"),
		JiraBaseURL:         os.Getenv("JIRA_BASE_URL"),
		JiraEmail:           os.Getenv("JIRA_EMAIL"),
		JiraAPIToken:        os.Getenv("JIRA_API_TOKEN"),
		JiraDefaultProject:  os.Getenv("JIRA_DEFAULT_PROJECT_KEY"),
		JiraStatusColumnMap: os.Getenv("JIRA_STATUS_COLUMN_MAP"),
		NtfyURL:             os.Getenv("NTFY_URL"),
		NtfyTopic:           os.Getenv("NTFY_TOPIC"),
		NtfyUsername:        os.Getenv("NTFY_USERNAME"),
		NtfyPassword:        os.Getenv("NTFY_PASSWORD"),
		AIProvider:          os.Getenv("AI_PROVIDER"),
		AIAPIKey:            os.Getenv("AI_API_KEY"),
		AIModel:             os.Getenv("AI_MODEL"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
