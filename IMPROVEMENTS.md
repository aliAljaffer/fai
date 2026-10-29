# Improvements

## Calendar

### Empty space below the month grid

The calendar card stops after the last week row and leaves a large blank area below it (October
2026 screenshot). Check the page shell's minimum height and `Calendar.tsx`'s wrapper
(`fai-calendar`, `min-w-[56rem]`, `height="auto"`) and make the page end with the grid instead of
reserving space under it.

### Week start preference

`firstDay` is hardcoded to `1` (Monday) in `frontend/src/pages/Calendar.tsx`. Add a preference for
the first day of the week, defaulting to Sunday for the Saudi work week (Sunday to Thursday). Use
the existing zustand `persist` pattern in `frontend/src/store/columnColors.ts` unless a per-user
settings table exists.

### Meeting summary from a past event, imported events included

Clicking an imported event only opens its link (`handleEventClick`). Offer "Add meeting summary"
from the event, with the event's date pre-filled, and keep the day-cell and header entry points.
Imported events need their date passed into `LogMeetingOutcomeDialog` as `occurredOn`.

### Date the summary from the meeting, not from insertion

Task extraction ignores when the meeting happened, so AI-created tasks come out undated. Pass the
meeting date (`occurred_on`, or `ends_at` when set) into the extraction prompt and instruct the
model to date suggested tasks relative to it ("next week", "follow up Friday") and to set
`due_date` from the meeting date.

## Jira export

### Issue types follow the subtask count

- More than 3 subtasks: export the parent as an Epic and the children as its subtasks.
- 3 or fewer subtasks: export the parent as a Task and the children as its subtasks.
- No subtasks: export a plain Task.

`jira.Client.CreateIssue` hardcodes issue type `Task` and sends only a summary. It needs an issue
type argument and a parent key, since Jira subtasks are created with
`parent: {key: <parent key>}`.

### A subtask cannot be exported without its parent

Export the parent first. If the parent has no Jira link, skip the child and report it, never
create the child alone. On failure of the parent, skip all of its children.

### Idempotent export and import

- `jira_links` already has `task_id` (primary key) and `jira_key` (unique). Use it as the dedupe
  key: a linked task is skipped, not created again.
- Import already skips by `jira_key` unless `force_update` is set. Extend that to subtask parents
  so a rerun does not duplicate or orphan children.
- Export a board or a selection as one job, and report created, skipped, and failed counts. Retry
  is safe: a second run links the same tasks to the same keys.
- Store the Jira key on the Fai task through `jira_links` and show it in the task sheet and the
  list row, so the link is visible without opening Jira.

## Workspaces

Workspaces are a separation layer over boards, lists, notes, and reminders. The separation is not
full segregation: any work item can be referenced from any workspace.

### Data model

- `workspaces`: `id`, `name`, `tag` (unique, 3 or 4 letters, case-insensitive), `created_at`.
- `workspace_id` on `boards`, `notes`, `reminders`, `calendar_sources`, and `meeting_outcomes`.
  Tasks, columns, attachments, and time entries inherit their workspace through their board or
  their owner.
- Seed one default workspace for existing rows. Both storage paths need the migration: a goose
  migration for Postgres and an additive `migrations_sqlite` file, so existing installs upgrade in
  place.
- Search and the command palette filter to the active workspace. Search results offer an "all
  workspaces" toggle instead of mixing results silently.
- A workspace switch lives in the nav next to the board switcher. Notes, reminders, and the
  calendar follow it, the same way the board switcher already scopes tasks.

### Task references

- `tasks.ref_num`: an integer from a per-workspace counter, unique on
  `(workspace_id, ref_num)`. Tasks and subtasks share one counter, and a reference does not
  distinguish them.
- A reference is `@TAG-123`, for example `@DEV-42`. It resolves at read time against the workspace
  tag plus the task number. It is never stored in the task title, so a tag change is one UPDATE on
  `workspaces` and no title edits. Titles stay as typed.
- Rename support: keep a `workspace_tag_aliases` row for each retired tag so old `@TAG-123` text
  keeps resolving. Drop an alias when a new workspace takes that tag.
- `@text` is the search form. It accepts spaces and is case-insensitive: `@cloud migration`
  searches for "cloud migration". The query runs on each keystroke after the `@`, so the picker
  narrows as the user types. Selecting a match replaces the typed query with the resolved
  `@TAG-123` reference, so stored text stays resolvable.
- Typing `@` or `#` opens the picker immediately, before any character follows it. The picker
  filters against the typed text on every keystroke, debounced the same way the Jira issue search
  already is, and closes on Escape or when the caret leaves the token.
- One parser serves every body: task description, note body, reminder title, meeting summary
  notes, and chat input. Render `@TAG-123` and resolved `@text` matches as links to the task
  sheet.
- Back-links: when a note, reminder, task, meeting summary, or chat message contains a reference,
  the task sheet lists the referring item. Reuse the rendering idea already used for note
  backlinks and `[[Title]]` wikilinks.

### Labels with `#label`

- Typing `#label` in a task body attaches that tag. Reuse `tags` and `task_tags` with a
  case-insensitive match, and create the tag when it does not exist. Suggest existing tags while
  typing, the same way Jira linking already suggests matches.
- A label cannot contain spaces. A typed space becomes a dash, so `#code review` is stored as
  `code-review`. The picker only offers dash-separated labels.
- Labels and workspace tags are separate concepts. Labels are global today and stay global.
- AI inference: ask the provider for 3 to 6 keyword labels from the task title and body, prefer
  existing tags, and show them as accept or reject chips in the task sheet. Never write a label
  without an explicit accept, and hide the feature when AI is unconfigured, following the current
  AI conventions.

### Open questions

- Chat is stateless today: `/api/ai/chat` persists nothing. Chat references need a persisted chat
  table before a chat message can back-link to a task.
- `@text` scope: search the active workspace first, then offer other workspaces as a second group
  so a cross-workspace reference is deliberate.
- The calendar ICS feed is one global token. Per-workspace feeds need a token per workspace, or one
  token with a workspace parameter.

## Assistant on Telegram or WhatsApp

Talk to Fai from a phone by text or voice, create tasks and reminders, and change task status.
Verdict from review: feasible. Telegram first, WhatsApp later. Most of the work already exists in
`POST /api/ai/chat` and `applyChatAction` (`backend/internal/httpapi/ai.go`).

### Transport

- Telegram first. BotFather token, no cost, voice notes, unrestricted outbound messages, and long
  polling. No public route, no webhook, no Caddy change.
- Use `getUpdates` long polling, not a webhook. Fai only makes outbound calls, it works behind NAT,
  and the offset prevents duplicate handling. One poller is enough for one owner. Switch to a
  webhook only if the backend runs more than one replica.
- WhatsApp later. Cloud API needs a Meta business account, a verified number, a public webhook
  (there is no polling), the 24 hour service window, and templates for proactive messages. The
  blockers are account setup and template rules, not code. Keep the transport behind a small
  interface so the same handler serves both.

### Bot placement

- Run the bot inside the existing Go binary from `backend/cmd/api/main.go`, reusing the store, the
  AI provider, and `applyChatAction`.
- Do not call `/api` over HTTP. Call the handler logic directly, so the bot needs no OIDC token or
  service account.
- Single owner: allowlist by both `from.id` and `chat.id`, accept private chats only, and silently
  drop everyone else. Keep the token in an env var and never log it.

### Conversation state

- `/api/ai/chat` is stateless and the client resends the last 10 messages. Keep the last 10
  messages per chat in an in-memory map keyed by chat ID. Loss on restart is acceptable for a chat.
  If persistence is needed later, add a `chat_messages` table (already an open item under
  Workspaces).

### Audio

- Telegram voice notes are OGG/Opus. Download with `getFile` and cap the length at 60 to 90 seconds.
- Use an audio-capable model through OpenCode Go instead of a dedicated STT endpoint. OpenCode Go
  has no `/audio/transcriptions` route. It does include MiMo-V2.5, MiMo-V2.5-Pro, MiMo-V2.6-Flash,
  MiMo-V2.6-Pro, and Gemini 4 Argon. OpenCode's own voice feature uses MiMo-V2.5 for this.
- Add a `Transcribe` method to the `ai.Provider` interface and both implementations. Post to
  `https://opencode.ai/inference/go/openai/v1/chat/completions` with model `mimo-v2.5` and a
  transcription-only system prompt: transcribe verbatim, remove filler words, do not answer
  questions, and output nothing when there is no speech. Set temperature 0.
- Wire shape: content parts
  `[{"type":"input_audio","input_audio":{"data":"<base64>","format":"mp3"}},{"type":"text","text":"<prompt>"}]`.
- Test the raw OGG first. If the gateway accepts OGG/Opus, skip ffmpeg. If it rejects the format,
  bundle ffmpeg in the backend image and convert to mono mp3, the same way OpenCode's voice code
  does. ffmpeg adds size to the image, so add it only when the test shows it is needed.
- Always echo `Heard: <transcript>` with the reply. A wrong transcript is the main failure mode and
  the echo makes it obvious. Fall back to "could not transcribe, send text" on failure.

### Provider configuration

- `backend/internal/ai/openaicompat.go` hardcodes `https://opencode.ai/zen/v1`. OpenCode Go uses
  `https://opencode.ai/inference/go/openai/v1`. Add a base URL setting, or an `opencode-go`
  provider case, and set `AI_MODEL=mimo-v2.5` for transcription.
- OpenCode Go is marketed for coding agents and monitors traffic for abuse. A personal task bot is
  not coding traffic. Use it if it works, but treat this as a risk. If Go rejects the traffic, fall
  back to Console pay-as-you-go or a dedicated Whisper API (Groq, OpenAI, or self-hosted
  faster-whisper). Keep transcription behind the `Provider` interface so a swap is one config
  change.

### Task status edits

- Status is the column. `POST /api/tasks/{id}/move` exists and `board_columns.is_done` already
  marks done columns. No schema change is needed.
- The chat prompt lists only task titles today, so the model cannot point at one task. Add the
  stable reference (`@DEV-42`) to the context.
- Add `find_task` and `move_task` actions. The model returns a task ID and a column name. The
  server resolves both, rejects an unknown column, and asks which task when the match is missing or
  ambiguous. Never let the model choose between near duplicates.
- Do not add a separate `set_completed` action. `completed_at` is set only by Jira import today,
  so stats ignore tasks finished in a done column. Fix the shared move path to set `completed_at`
  on entry to a done column and clear it on exit, so the web UI and the bot agree.
- One action per reply is enough. Drop the idea of a JSON array of actions.
- Confirm before writes that are not creates. Hold `move_task` as pending and show Telegram inline
  buttons (Confirm, Cancel) on `callback_query`. Expire pending actions after a few minutes. Always
  confirm voice-originated edits because the transcript can be wrong. Creates stay immediate,
  matching current behavior.

### Outbound reminders

- Add Telegram next to ntfy in `backend/internal/reminders/scheduler.go`, using a small client
  modeled on `backend/internal/ntfy/client.go`.
- Choose the channel with a setting instead of sending to every channel, so reminders do not
  arrive twice.

### Milestones

1. Text-only Telegram over long polling: allowlist, reuse the chat prompt and `applyChatAction`,
   create task and create reminder only. Reply with what was created.
2. Telegram reminder channel.
3. Voice, after the OGG smoke test, with the transcript echo.
4. Find, move, and complete, with confirmation buttons. Fix `completed_at` in the shared move path.
5. WhatsApp Cloud API behind the same transport interface, if it is still needed.

### Risks

- Transcription quality on short voice notes with names and references. The echo and the confirm
  step bound the damage.
- Prompt injection from task titles in the chat context. Keep the action set small and validate
  every field server side.
- Telegram updates are at least once. Persist the `getUpdates` offset after each processed update
  so a restart does not create a task twice (moves are close to idempotent, creates are not).
- Two LLM calls per voice message. Latency is acceptable for chat.
