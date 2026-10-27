// Package migrationssqlite embeds the SQLite schema so the binary can apply it
// itself on first run — unlike the Postgres path, a file-based SQLite deployment
// has no separate migration-runner step to wire into docker-compose.
package migrationssqlite

import _ "embed"

//go:embed 001_init.sql
var Schema string
