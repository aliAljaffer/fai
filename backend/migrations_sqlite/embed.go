// Package migrationssqlite embeds the SQLite schema so the binary can apply it
// itself — a file-based SQLite deployment has no separate migration-runner step
// to wire into docker-compose.
package migrationssqlite

import _ "embed"

//go:embed 001_init.sql
var Schema string

//go:embed 002_attachments.sql
var Attachments string
