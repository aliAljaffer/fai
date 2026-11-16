# Changelog

## v0.3.0

### Changes

- Make labels belong to a workspace: scope the tags table, API, backups and frontend label queries by workspace, and split existing shared labels per workspace.
- Collapse reminders into a rail, move board actions into the board menu, show active filters in the toolbar, and drop the coloured card side bar
- Redesign task cards and the sidebar: flat status-accented cards, a label gallery that filters the board and list in place, a notes count endpoint, and an idle timer panel
- Move the app to a sidebar layout with a shared page header, resizable board columns, and task details under the clicked card
- Apply modern web guidance: route code-splitting, IANA timezones, report-only CSP, and native form, disclosure and progress fixes
- Parse repeat phrases in the natural-language reminder and task inputs
- Add repeating reminders and tasks, timer chat actions, saved Telegram history, and reminders in the calendar feed
- ci: attribute manifest commits to update-manifests-homelab
- List every arc task with its step count in the chat confirmation
- Accept Telegram voice notes up to 10 minutes and transcribe long notes in 60-second parts
- Let the chat agent edit, complete, delete, tag, comment on, and log time for tasks, and manage reminders and notes
- Add quarantine update instructions [skip ci]
- Use the owner timezone for Telegram relative dates
- Clear stale Telegram confirmations on new messages
- Document confirmation expiry
- Confirm every mutating chat action
- Document OpenCode Go primary with free fallback
- Add AI model fallback and OpenCode Go session header
- Add OpenAI-compatible speech-to-text
- Parallelize independent I/O in hot paths
- removed uppercasing where not needed, greeting is now date-based
- Make the whole board card draggable and fix the drag preview loop
- Fix board drag placement and add drag tests
- Send macOS users to the download page instead of the unsigned updater [skip ci]
- Generate release notes automatically and sign macOS builds [skip ci]
- Add the changelog

### Dependencies

- Bump the desktop version to 0.3.0
- Bump GitHub Actions off Node 20 [skip ci]
- Group Dependabot updates per ecosystem to reduce PR count

**Full changelog:** https://github.com/aliAljaffer/fai/compare/v0.2.0...v0.3.0

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
