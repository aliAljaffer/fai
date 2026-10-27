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

	server := &httpapi.Server{Q: sqlc.New(pool), Pool: pool}
	router := httpapi.NewRouter(server)

	addr := ":" + cfg.Port
	log.Printf("fai backend listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
