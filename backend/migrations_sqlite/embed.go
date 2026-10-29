// Package migrationssqlite embeds the SQLite schema so the binary can apply it
// itself — a file-based SQLite deployment has no separate migration-runner step
// to wire into docker-compose.
package migrationssqlite

import _ "embed"

//go:embed 001_init.sql
var Schema string

//go:embed 002_attachments.sql
var Attachments string

//go:embed 003_calendar_sources.sql
var CalendarSources string

//go:embed 004_calendar_feed_cache.sql
var CalendarFeedCache string

//go:embed 005_workspaces.sql
var Workspaces string

//go:embed 006_task_comments.sql
var TaskComments string

//go:embed 007_ai_settings.sql
var AISettings string

//go:embed 008_appearance.sql
var Appearance string

//go:embed 009_chat_messages.sql
var ChatMessages string
