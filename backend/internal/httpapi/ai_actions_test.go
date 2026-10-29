package httpapi

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func actionTestStore(t *testing.T) *lite.Store {
	t.Helper()
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return st
}

func columnID(t *testing.T, st *lite.Store, name string) int64 {
	t.Helper()
	columns, err := st.ListColumnsByBoard(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range columns {
		if column.Name == name {
			return column.ID
		}
	}
	t.Fatalf("column %q not found", name)
	return 0
}

func TestChatMoveTaskIsPendingUntilConfirmed(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	todoID := columnID(t, st, "To Do")
	doneID := columnID(t, st, "Done")
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: todoID, Title: "Refresh the Cloudflare token", Priority: "medium", Position: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	provider := &fakeAI{reply: `Moving it.
ACTION: {"type":"move_task","task":"@` + task.Ref + `","column":"Done"}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "move it to done"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending == nil {
		t.Fatalf("pending = nil, reply = %q", outcome.Reply)
	}
	if outcome.Pending.ColumnID != doneID || outcome.Pending.ColumnName != "Done" {
		t.Fatalf("pending = %#v", outcome.Pending)
	}
	if !strings.Contains(outcome.Reply, "Move @"+task.Ref) {
		t.Errorf("reply = %q", outcome.Reply)
	}

	unchanged, err := st.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.ColumnID != todoID {
		t.Fatalf("task moved before confirmation: column %d", unchanged.ColumnID)
	}

	confirmation, err := server.ConfirmChatAction(ctx, *outcome.Pending)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(confirmation, "Moved") {
		t.Errorf("confirmation = %q", confirmation)
	}
	moved, err := st.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.ColumnID != doneID {
		t.Fatalf("column = %d, want %d", moved.ColumnID, doneID)
	}
}

func TestChatFindTaskUsesRef(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: columnID(t, st, "To Do"), Title: "Refresh the Cloudflare token", Priority: "medium", Position: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	provider := &fakeAI{reply: `Here it is.
ACTION: {"type":"find_task","task":"@` + task.Ref + `"}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "where is my token task"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending != nil {
		t.Fatal("find_task created a pending action")
	}
	if !strings.Contains(outcome.Reply, "@"+task.Ref) || !strings.Contains(outcome.Reply, "To Do") {
		t.Fatalf("reply = %q", outcome.Reply)
	}
}

func TestChatFindTaskAcceptsNumericID(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: columnID(t, st, "To Do"), Title: "Refresh the Cloudflare token", Priority: "medium", Position: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	provider := &fakeAI{reply: `ACTION: {"type":"find_task","task_id":` + strconv.FormatInt(task.ID, 10) + `}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "where is my token task"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(outcome.Reply, "@"+task.Ref) {
		t.Fatalf("reply = %q", outcome.Reply)
	}
}

func TestChatMoveTaskRejectsUnknownColumn(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: columnID(t, st, "To Do"), Title: "Refresh the Cloudflare token", Priority: "medium", Position: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	provider := &fakeAI{reply: `ACTION: {"type":"move_task","task":"@` + task.Ref + `","column":"Shipped"}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "move it to shipped"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending != nil {
		t.Fatal("unknown column produced a pending action")
	}
	if !strings.Contains(outcome.Reply, "could not find a column named Shipped") || !strings.Contains(outcome.Reply, "Done") {
		t.Fatalf("reply = %q", outcome.Reply)
	}
}

func TestChatMoveTaskAsksForMissingOrAmbiguousTask(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	todoID := columnID(t, st, "To Do")
	for _, title := range []string{"Fix the bug", "Fix the build"} {
		if _, err := st.CreateTask(ctx, store.CreateTaskParams{
			BoardID: 1, ColumnID: todoID, Title: title, Priority: "medium", Position: 1024,
		}); err != nil {
			t.Fatal(err)
		}
	}

	server := &Server{Store: st, AI: &fakeAI{reply: `ACTION: {"type":"move_task","task":"@FAI-999","column":"Done"}`}}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "move that task"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending != nil || !strings.Contains(outcome.Reply, "could not find that task") {
		t.Fatalf("missing task outcome = %#v", outcome)
	}

	server.AI = &fakeAI{reply: `ACTION: {"type":"move_task","task":"Fix","column":"Done"}`}
	outcome, err = server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "move the fix task"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending != nil || !strings.Contains(outcome.Reply, "Which task") {
		t.Fatalf("ambiguous task outcome = %#v", outcome)
	}
}

func TestFaiContextIncludesTaskRefs(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: columnID(t, st, "To Do"), Title: "Refresh the Cloudflare token", Priority: "medium", Position: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: st}
	contextText := server.faiContext(ctx)
	if !strings.Contains(contextText, "@"+task.Ref+" Refresh the Cloudflare token") {
		t.Fatalf("context = %q", contextText)
	}
}
