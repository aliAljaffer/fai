package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/auth"
	"github.com/alialjaffer/fai/backend/internal/config"
	"github.com/alialjaffer/fai/backend/internal/httpapi"
	"github.com/alialjaffer/fai/backend/internal/jira"
	"github.com/alialjaffer/fai/backend/internal/ntfy"
	"github.com/alialjaffer/fai/backend/internal/reminders"
	"github.com/alialjaffer/fai/backend/internal/store/pg"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	var authMiddleware func(http.Handler) http.Handler
	if cfg.DisableAuth {
		log.Println("WARNING: DISABLE_AUTH=true — authentication is OFF. Never set this in production.")
	} else {
		// Fail-closed: auth is on by default, so a missing issuer/audience is a
		// startup error, not a silent bypass.
		if cfg.OIDCIssuerURL == "" || cfg.OIDCAudience == "" {
			log.Fatal("OIDC_ISSUER_URL and OIDC_AUDIENCE are required unless DISABLE_AUTH=true")
		}
		authMiddleware = auth.MustMiddleware(context.Background(), cfg.OIDCIssuerURL, cfg.OIDCAudience)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	st := pg.New(pool)
	defer st.Close()

	jiraClient := jira.New(cfg.JiraBaseURL, cfg.JiraEmail, cfg.JiraAPIToken)
	if jiraClient == nil {
		log.Println("JIRA_BASE_URL/EMAIL/API_TOKEN unset — Jira import disabled")
	}
	aiProvider := ai.NewProvider(cfg.AIProvider, cfg.AIAPIKey, cfg.AIModel)
	if aiProvider == nil {
		log.Println("AI_API_KEY unset — AI features disabled")
	}
	server := &httpapi.Server{
		Store:               st,
		Jira:                jiraClient,
		JiraDefaultProject:  cfg.JiraDefaultProject,
		JiraStatusColumnMap: jira.ParseStatusColumnMap(cfg.JiraStatusColumnMap),
		AI:                  aiProvider,
		AuthMiddleware:      authMiddleware,
	}
	router := httpapi.NewRouter(server)

	ntfyClient := ntfy.New(cfg.NtfyURL, cfg.NtfyTopic, cfg.NtfyUsername, cfg.NtfyPassword)
	if ntfyClient == nil {
		log.Println("NTFY_URL/NTFY_TOPIC unset — reminder push notifications disabled")
	}
	scheduler := &reminders.Scheduler{Store: st, Ntfy: ntfyClient}
	go scheduler.Run(context.Background())

	addr := ":" + cfg.Port
	log.Printf("fai backend listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
