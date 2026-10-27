// Package migrations embeds the Postgres migrations so the binary can apply them
// itself, the same way the SQLite path does — a deployment otherwise has no
// migration-runner step to wire in.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
