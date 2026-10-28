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
- `@text` is the search form: it queries `/api/search` with the text and shows a picker of matches.
  Selecting a match inserts the `@TAG-123` reference, so stored text is always resolvable.
- One parser serves every body: task description, note body, reminder title, meeting summary
  notes, and chat input. Render `@TAG-123` and `@text` as links to the task sheet.
- Back-links: when a note, reminder, task, meeting summary, or chat message contains a reference,
  the task sheet lists the referring item. Reuse the rendering idea already used for note
  backlinks and `[[Title]]` wikilinks.

### Labels with `#label`

- Typing `#label` in a task body attaches that tag. Reuse `tags` and `task_tags` with a
  case-insensitive match, and create the tag when it does not exist. Suggest existing tags while
  typing, the same way Jira linking already suggests matches.
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
