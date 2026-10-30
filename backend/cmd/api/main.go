package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/auth"
	"github.com/alialjaffer/fai/backend/internal/config"
	"github.com/alialjaffer/fai/backend/internal/httpapi"
	"github.com/alialjaffer/fai/backend/internal/jira"
	"github.com/alialjaffer/fai/backend/internal/mailer"
	"github.com/alialjaffer/fai/backend/internal/metrics"
	"github.com/alialjaffer/fai/backend/internal/migrate"
	"github.com/alialjaffer/fai/backend/internal/ntfy"
	"github.com/alialjaffer/fai/backend/internal/reminders"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
	"github.com/alialjaffer/fai/backend/internal/store/pg"
	"github.com/alialjaffer/fai/backend/internal/telegram"
	"github.com/alialjaffer/fai/backend/internal/transcribe"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	var authMiddleware func(http.Handler) http.Handler
	switch {
	case cfg.DisableAuth:
		log.Println("WARNING: DISABLE_AUTH=true: authentication is OFF. Never set this in production.")
	case cfg.AuthMode == "proxy-header":
		log.Println("AUTH_MODE=proxy-header: trusting X-Forwarded-User/-Email from a fronting oauth2-proxy. The backend MUST be unreachable except through that proxy.")
		authMiddleware = auth.ProxyHeaderMiddleware
	default:
		// Fail-closed: auth is on by default, so a missing issuer/audience is a
		// startup error, not a silent bypass.
		if cfg.OIDCIssuerURL == "" || cfg.OIDCAudience == "" {
			log.Fatal("OIDC_ISSUER_URL and OIDC_AUDIENCE are required unless DISABLE_AUTH=true or AUTH_MODE=proxy-header")
		}
		authMiddleware = auth.MustMiddleware(context.Background(), cfg.OIDCIssuerURL, cfg.OIDCAudience)
	}

	var st store.Store
	if cfg.IsSQLite() {
		s, err := lite.New(cfg.SQLitePath())
		if err != nil {
			log.Fatalf("opening sqlite database: %v", err)
		}
		st = s
	} else {
		if err := migrate.Postgres(context.Background(), cfg.DatabaseURL); err != nil {
			log.Fatalf("migrating database: %v", err)
		}
		pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("connecting to database: %v", err)
		}
		st = pg.New(pool)
	}
	defer st.Close()

	jiraClient := jira.New(cfg.JiraBaseURL, cfg.JiraEmail, cfg.JiraAPIToken)
	if jiraClient == nil {
		log.Println("JIRA_BASE_URL/EMAIL/API_TOKEN unset: Jira import disabled")
	}
	aiProvider := ai.NewProvider(cfg.AIProvider, cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel)
	if aiProvider == nil {
		log.Println("AI_API_KEY unset: AI features disabled")
	}
	server := &httpapi.Server{
		Store:                st,
		Jira:                 jiraClient,
		JiraDefaultProject:   cfg.JiraDefaultProject,
		JiraStatusColumnMap:  jira.ParseStatusColumnMap(cfg.JiraStatusColumnMap),
		JiraEpicIssueType:    cfg.JiraEpicIssueType,
		JiraTaskIssueType:    cfg.JiraTaskIssueType,
		JiraSubtaskIssueType: cfg.JiraSubtaskIssueType,
		JiraEpicLinkField:    cfg.JiraEpicLinkField,
		AI:                   aiProvider,
		AIProvider:           cfg.AIProvider,
		AIBaseURL:            cfg.AIBaseURL,
		AIModel:              cfg.AIModel,
		AIAPIKey:             cfg.AIAPIKey,
		ICSToken:             cfg.ICSToken,
		PublicURL:            cfg.PublicURL,
		AttachmentsDir:       cfg.AttachmentsDir,
		StaticDir:            cfg.StaticDir,
		AuthMiddleware:       authMiddleware,
		Metrics:              metrics.New(st),
	}
	if cfg.ElevenLabsAPIKey != "" {
		server.Transcriber = transcribe.NewElevenLabs(cfg.ElevenLabsAPIKey, cfg.ElevenLabsSTTModel)
		log.Println("ELEVENLABS_API_KEY set: voice notes use ElevenLabs speech-to-text")
	}
	router := httpapi.NewRouter(server)

	telegramClient := telegram.New(cfg.TelegramBotToken)
	var telegramBot *telegram.Bot
	switch {
	case telegramClient == nil:
		log.Println("TELEGRAM_BOT_TOKEN unset, Telegram assistant disabled")
	case cfg.TelegramAllowedUserID == 0 || cfg.TelegramAllowedChatID == 0:
		log.Println("TELEGRAM_ALLOWED_USER_ID/TELEGRAM_ALLOWED_CHAT_ID unset, Telegram assistant disabled")
	default:
		telegramBot = telegram.NewBot(telegramClient, cfg.TelegramAllowedUserID, cfg.TelegramAllowedChatID, server).
			WithOffsetStore(telegram.FileOffsetStore{Path: cfg.TelegramStateFile})
	}

	ntfyClient := ntfy.New(cfg.NtfyURL, cfg.NtfyTopic, cfg.NtfyUsername, cfg.NtfyPassword)
	if ntfyClient == nil {
		log.Println("NTFY_URL/NTFY_TOPIC unset: reminder push notifications disabled")
	}
	if cfg.AttachmentsDir == "" {
		log.Println("ATTACHMENTS_DIR unset: task/note attachments disabled")
	}
	mailClient := mailer.New(cfg.SMTPAddr, cfg.SMTPFrom, cfg.SMTPTo)
	if mailClient == nil {
		log.Println("SMTP_ADDR/SMTP_FROM/SMTP_TO unset: email notifications disabled")
	}
	telegramNotifier := telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramAllowedChatID)
	scheduler := &reminders.Scheduler{
		Store: st, Ntfy: ntfyClient, Mail: mailClient,
		Telegram: telegramNotifier, Channel: cfg.ReminderChannel,
	}
	go scheduler.Run(context.Background())
	go server.RunCalendarSync(context.Background())
	if telegramBot != nil {
		go telegramBot.Run(context.Background())
	}

	addr := net.JoinHostPort(cfg.BindAddr, cfg.Port)
	if cfg.MetricsAddr != "" {
		go func() {
			log.Printf("fai metrics listening on %s", cfg.MetricsAddr)
			if err := http.ListenAndServe(cfg.MetricsAddr, server.Metrics.Handler()); err != nil {
				log.Printf("metrics server stopped: %v", err)
			}
		}()
	}
	log.Printf("fai backend listening on %s", addr)
	// ReadHeaderTimeout bounds unauthenticated slow-header requests; the read and
	// write timeouts stay unset because AI calls can legitimately run for a minute.
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
