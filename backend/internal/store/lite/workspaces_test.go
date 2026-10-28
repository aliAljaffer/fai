package lite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return st
}

func mustWorkspace(t *testing.T, st *Store, name, tag string) store.Workspace {
	t.Helper()
	w, err := st.CreateWorkspace(context.Background(), name, tag)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func mustBoard(t *testing.T, st *Store, name string, workspaceID int64) store.Board {
	t.Helper()
	b, err := st.CreateBoard(context.Background(), name, workspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateColumn(context.Background(), b.ID, "To Do", 0, false); err != nil {
		t.Fatal(err)
	}
	return b
}

func columnOf(t *testing.T, st *Store, boardID int64) int64 {
	t.Helper()
	cols, err := st.ListColumnsByBoard(context.Background(), boardID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) == 0 {
		t.Fatal("board has no columns")
	}
	return cols[0].ID
}

func mustTask(t *testing.T, st *Store, boardID, columnID int64, title, body string) store.Task {
	t.Helper()
	task, err := st.CreateTask(context.Background(), store.CreateTaskParams{
		BoardID: boardID, ColumnID: columnID, Title: title, DescriptionMd: body, Priority: "medium", Position: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestRefNumAllocationPerWorkspace(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	workspaces, err := st.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0].Tag != "FAI" || workspaces[0].Name != "Personal" {
		t.Fatalf("seed workspaces = %#v", workspaces)
	}

	personal := mustBoard(t, st, "Personal", 0)
	dev := mustWorkspace(t, st, "Dev", "dev")
	devBoard := mustBoard(t, st, "Dev", dev.ID)
	if dev.Tag != "DEV" {
		t.Errorf("tag stored as %q", dev.Tag)
	}

	first := mustTask(t, st, personal.ID, columnOf(t, st, personal.ID), "One", "")
	second := mustTask(t, st, personal.ID, columnOf(t, st, personal.ID), "Two", "")
	other := mustTask(t, st, devBoard.ID, columnOf(t, st, devBoard.ID), "Other", "")

	if first.Ref != "FAI-1" || second.Ref != "FAI-2" {
		t.Errorf("personal refs = %q %q", first.Ref, second.Ref)
	}
	if other.Ref != "DEV-1" {
		t.Errorf("dev ref = %q", other.Ref)
	}
	if first.WorkspaceID != workspaces[0].ID || other.WorkspaceID != dev.ID {
		t.Errorf("workspace ids = %d %d", first.WorkspaceID, other.WorkspaceID)
	}

	sub, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: personal.ID, ColumnID: columnOf(t, st, personal.ID), Title: "Sub",
		Priority: "medium", Position: 2048, ParentTaskID: &first.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sub.RefNum != second.RefNum+1 {
		t.Errorf("subtask shares the counter, got %d after %d", sub.RefNum, second.RefNum)
	}
}

func TestTagRenameKeepsAliasResolving(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	dev := mustWorkspace(t, st, "Dev", "DEV")
	board := mustBoard(t, st, "Dev", dev.ID)
	task := mustTask(t, st, board.ID, columnOf(t, st, board.ID), "Ship it", "")

	rename := "WRK"
	if _, err := st.UpdateWorkspace(ctx, dev.ID, nil, &rename); err != nil {
		t.Fatal(err)
	}

	for _, query := range []string{"DEV-1", "wrk-1"} {
		refs, err := st.ResolveRefs(ctx, query, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 1 || refs[0].ID != task.ID {
			t.Fatalf("resolve %q = %#v", query, refs)
		}
		if refs[0].Ref != "WRK-1" || refs[0].WorkspaceTag != "WRK" {
			t.Errorf("resolve %q ref = %q tag = %q", query, refs[0].Ref, refs[0].WorkspaceTag)
		}
	}

	if _, _, ok := store.ParseTaskRef("DEV-1"); !ok {
		t.Error("DEV-1 should parse as a reference")
	}

	taken := "FAI"
	if _, err := st.UpdateWorkspace(ctx, dev.ID, nil, &taken); err == nil {
		t.Error("renaming to a live tag should fail")
	}
}

func TestWorkspaceScopingOnLists(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	dev := mustWorkspace(t, st, "Dev", "DEV")
	personalBoard := mustBoard(t, st, "Personal", 0)
	devBoard := mustBoard(t, st, "Dev", dev.ID)
	if _, err := st.CreateNote(ctx, "Personal note", "body", nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateNote(ctx, "Dev note", "body", nil, dev.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateReminder(ctx, "Personal reminder", time.Now().Add(time.Hour), nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateReminder(ctx, "Dev reminder", time.Now().Add(time.Hour), nil, dev.ID); err != nil {
		t.Fatal(err)
	}

	boards, err := st.ListBoards(ctx, &dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 1 || boards[0].ID != devBoard.ID || boards[0].WorkspaceID != dev.ID {
		t.Errorf("scoped boards = %#v", boards)
	}

	notes, err := st.ListNotes(ctx, &dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Title != "Dev note" || notes[0].WorkspaceID != dev.ID {
		t.Errorf("scoped notes = %#v", notes)
	}

	reminders, err := st.ListReminders(ctx, &dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 1 || reminders[0].Title != "Dev reminder" {
		t.Errorf("scoped reminders = %#v", reminders)
	}

	all, err := st.ListNotes(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("unscoped notes = %d", len(all))
	}

	sources, err := st.ListCalendarSources(ctx, &dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Errorf("dev sources = %#v", sources)
	}
	if _, err := st.CreateCalendarSource(ctx, "Dev feed", "https://example.com/a.ics", "#2563eb", dev.ID); err != nil {
		t.Fatal(err)
	}
	sources, err = st.ListCalendarSources(ctx, &dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].WorkspaceID != dev.ID {
		t.Errorf("dev sources = %#v", sources)
	}

	if _, err := st.CreateMeetingOutcome(ctx, "2026-10-20", "Personal sync", "notes", nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateMeetingOutcome(ctx, "2026-10-21", "Dev sync", "notes", nil, dev.ID); err != nil {
		t.Fatal(err)
	}
	outcomes, err := st.ListMeetingOutcomes(ctx, nil, &dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].Title != "Dev sync" || outcomes[0].WorkspaceID != dev.ID {
		t.Errorf("scoped outcomes = %#v", outcomes)
	}

	if personalBoard.ID == devBoard.ID {
		t.Error("boards should differ")
	}
}

func TestTaskReferencesScan(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	board := mustBoard(t, st, "Personal", 0)
	columnID := columnOf(t, st, board.ID)
	target := mustTask(t, st, board.ID, columnID, "Target", "the one")
	mustTask(t, st, board.ID, columnID, "Mentions", "blocked by @fai-1 until friday")
	if _, err := st.CreateNote(ctx, "Note mention", "see @FAI-1", nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateNote(ctx, "Title mention @FAI-1", "no ref here", nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateReminder(ctx, "Follow up on @FAI-1", time.Now().Add(time.Hour), nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateNote(ctx, "Unrelated", "nothing", nil, 0); err != nil {
		t.Fatal(err)
	}

	refs, err := st.ListTaskReferences(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 4 {
		t.Fatalf("references = %#v", refs)
	}
	kinds := map[string]int{}
	for _, ref := range refs {
		kinds[ref.Kind]++
		if ref.WorkspaceTag != "FAI" {
			t.Errorf("backlink tag = %q", ref.WorkspaceTag)
		}
	}
	if kinds["task"] != 1 || kinds["note"] != 2 || kinds["reminder"] != 1 {
		t.Errorf("kinds = %#v", kinds)
	}

	if _, err := st.ListTaskReferences(ctx, 9999); err != store.ErrNotFound {
		t.Errorf("missing task error = %v", err)
	}
}

func TestBodyLabels(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	existing, err := st.CreateTag(ctx, "Bug", "#ff0000")
	if err != nil {
		t.Fatal(err)
	}
	board := mustBoard(t, st, "Personal", 0)
	columnID := columnOf(t, st, board.ID)

	task := mustTask(t, st, board.ID, columnID, "Labeled", "fix #bug and #code-review then # spaced")
	tags, err := st.ListTagsForTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]int64{}
	for _, tag := range tags {
		names[tag.Name] = tag.ID
	}
	if len(tags) != 2 {
		t.Fatalf("tags = %#v", tags)
	}
	if names["Bug"] != existing.ID {
		t.Errorf("existing tag should be reused, got %#v", names)
	}
	if _, ok := names["code-review"]; !ok {
		t.Errorf("dash token missing: %#v", names)
	}

	explicit, err := st.CreateTag(ctx, "manual", "#00ff00")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddTaskTag(ctx, task.ID, explicit.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := st.UpdateTask(ctx, task.ID, task.Title, "no labels here", "medium", nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Ref != task.Ref {
		t.Errorf("ref changed: %q", updated.Ref)
	}
	tags, err = st.ListTagsForTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 3 {
		t.Errorf("labels should stay attached, got %#v", tags)
	}

	newTag, err := st.CreateTag(ctx, "Release", "#0000ff")
	if err != nil {
		t.Fatal(err)
	}
	updated, err = st.UpdateTask(ctx, task.ID, task.Title, "also #release", "medium", nil)
	if err != nil {
		t.Fatal(err)
	}
	tags, err = st.ListTagsForTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tag := range tags {
		if tag.ID == newTag.ID {
			found = true
		}
	}
	if !found || len(tags) != 4 {
		t.Errorf("additive labels = %#v", tags)
	}
}

func TestConcurrentRefAllocation(t *testing.T) {
	st := newTestStore(t)
	board := mustBoard(t, st, "Personal", 0)
	columnID := columnOf(t, st, board.ID)

	const count = 8
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		go func() {
			_, err := st.CreateTask(context.Background(), store.CreateTaskParams{
				BoardID: board.ID, ColumnID: columnID, Title: "Parallel", Priority: "medium", Position: 1024,
			})
			errs <- err
		}()
	}
	for i := 0; i < count; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}

	tasks, err := st.ListTasks(context.Background(), store.ListTasksParams{BoardID: &board.ID})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	for _, task := range tasks {
		if seen[task.RefNum] {
			t.Fatalf("duplicate ref_num %d", task.RefNum)
		}
		seen[task.RefNum] = true
	}
	if len(seen) != count {
		t.Errorf("refs = %d", len(seen))
	}
}
