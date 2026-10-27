package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/alialjaffer/fai/backend/internal/config"
	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/httpapi"
	"github.com/alialjaffer/fai/backend/internal/ntfy"
	"github.com/alialjaffer/fai/backend/internal/reminders"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	if cfg.DisableAuth {
		log.Println("WARNING: DISABLE_AUTH=true — authentication is OFF. Never set this in production.")
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)
	server := &httpapi.Server{Q: queries, Pool: pool}
	router := httpapi.NewRouter(server)

	ntfyClient := ntfy.New(cfg.NtfyURL, cfg.NtfyTopic, cfg.NtfyUsername, cfg.NtfyPassword)
	if ntfyClient == nil {
		log.Println("NTFY_URL/NTFY_TOPIC unset — reminder push notifications disabled")
	}
	scheduler := &reminders.Scheduler{Q: queries, Ntfy: ntfyClient}
	go scheduler.Run(context.Background())

	addr := ":" + cfg.Port
	log.Printf("fai backend listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
