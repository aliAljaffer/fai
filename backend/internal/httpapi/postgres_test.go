package httpapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alialjaffer/fai/backend/internal/migrate"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
	"github.com/alialjaffer/fai/backend/internal/store/pg"
)

// testStore returns a SQLite store by default. Set FAI_TEST_DATABASE_URL to a
// Postgres connection string to run the same tests against Postgres, each test
// in its own schema so runs cannot see each other's rows.
func testStore(t *testing.T) store.Store {
	t.Helper()
	base := os.Getenv("FAI_TEST_DATABASE_URL")
	if base == "" {
		st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(st.Close)
		return st
	}
	return postgresStore(t, base)
}

func postgresStore(t *testing.T, base string) store.Store {
	t.Helper()
	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())

	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatalf("connecting to postgres: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatalf("creating schema: %v", err)
	}
	// Registered before the store so it runs last, after the store's pool closes.
	t.Cleanup(func() {
		defer admin.Close()
		if _, err := admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("dropping schema %s: %v", schema, err)
		}
	})

	dsn := withSearchPath(base, schema)
	if err := migrate.Postgres(ctx, dsn); err != nil {
		t.Fatalf("migrating postgres: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting to postgres schema: %v", err)
	}
	st := pg.New(pool)
	t.Cleanup(st.Close)
	return st
}

func TestPostgresMigrationsAreIdempotent(t *testing.T) {
	base := os.Getenv("FAI_TEST_DATABASE_URL")
	if base == "" {
		t.Skip("set FAI_TEST_DATABASE_URL to run Postgres migrations")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("test_migrate_%d", time.Now().UnixNano())
	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)

	dsn := withSearchPath(base, schema)
	if err := migrate.Postgres(ctx, dsn); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := migrate.Postgres(ctx, dsn); err != nil {
		t.Fatalf("second migration: %v", err)
	}
}

// withSearchPath pins every connection in the pool to one schema, so tests share
// a database without sharing rows.
func withSearchPath(base, schema string) string {
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return base + separator + "options=-c%20search_path%3D" + schema
}
