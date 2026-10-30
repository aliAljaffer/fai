# Changelog

## v0.2.0

### AI and chat

- Chat creates tasks with steps, and arcs with nested tasks. Every ACTION line runs and Telegram messages are logged.
- Create tasks from natural language, with bulk and inline label suggestions.
- Note plans and meeting summaries, including summary arcs with steps and calendar links.
- AI provider settings stored in the database for Anthropic, OpenAI and compatible models.
- Telegram assistant.

### Voice

- ElevenLabs speech-to-text for voice notes, with audio format normalization and a fallback.

### Board and list

- Redesigned expanded task view for the board and the list.
- Task editing in place, nested step indentation and a pauseable undo toast.
- Drag-and-drop column reordering, task drop previews, and board and column renaming.
- New task dialog in the list view, plus task comments.
- List column picker with last updated, sort by updated, and Jira columns.
- Arc, Task and Step types with status icons and Jira type mapping.
- Reminders can be completed into the Done column.

### Jira and calendar

- Jira imports run in the background with app-wide progress. They are scoped to your assignee and you pick the target board.
- Imported descriptions and assignee comments, backfilled completion dates, and linked key mentions.
- Bulk Jira export from the task list.
- Calendars dialog stays within the viewport.

### Live updates and operations

- Server-sent events push every change to every open client.
- Full instance backup and restore.
- Prometheus metrics and an installation guide.
- Cluster images published to Harbor.

### Desktop

- Electron auto-updates, with branding and the app version in settings.
- ICO, PNG and Apple touch favicons.

### Fixes

- Board drag placement, new task position, and command palette overflow.
- Base UI menu and select label errors.
- Keyboard access, responsive navigation, form labels and page semantics.
- Timer polls every 30 seconds instead of every second.
- Jira numeric comment ids no longer fail the import.
- Notes list is shown again after a delete.

### Dependencies

- Eight dependency bumps across GitHub Actions, FullCalendar, Recharts and Electron.
