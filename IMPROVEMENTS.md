# Improvements

## List view

- Show tasks that have subtasks in the table with a chevron to expand and collapse them.
  Subtasks appear only under their parent task.
- If a task is linked to Jira, show the issue key (for example `DEV-111`) at the end of its
  row.
- Give Done, To Do, and In Progress distinct status colors. Apply the color to the status
  only, not the whole row. Let users edit these colors in Settings.
- Add a toggle to hide Done tasks. Done subtasks cannot be hidden; they stay visible and
  render struck through.

## Jira import

- Import Jira subtasks under their parent task instead of as separate top-level tasks.

  Cause: the importer requests only `summary,status` and never sets `parent_task_id`, so
  every Jira issue becomes a root task.

  Fix:
  1. Request `parent,subtasks` in the Jira search and read the parent key from each issue.
  2. Run the import in two passes. Import parents first, then child issues with
     `parent_task_id` set to the Fai task linked to the parent key.
  3. Look up the parent in the current import run first, then in existing Jira links, so a
     rerun does not duplicate or orphan subtasks.
  4. On force update, also correct a linked subtask's parent when Jira moved it.
  5. Skip a subtask when its parent is not in Fai, and add it to the skipped count instead
     of creating an orphan.
  6. Flatten Jira subtasks deeper than one level under the top imported ancestor.

## Calendar

- Make it possible to add a normal event. Clicking a day currently opens "Add meeting
  summary" straight away, which blocks that.
- Link an Outlook calendar and sync it one way, Outlook to Fai, so events stay current.
- Add meeting summaries only to meetings that already happened. A future meeting gets no
  summary prompt.

## Board

- Animate the destination column during drag and drop, not only the column the task came
  from. The current behavior makes it hard to see where the task will land.
