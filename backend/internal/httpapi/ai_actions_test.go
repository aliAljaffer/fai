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

func TestChatCreateTaskStepsNestUnderTask(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	provider := &fakeAI{reply: `I will create one task with five steps.
ACTION: {"type":"create_task","title":"Update homelab TalosOS version","steps":["Update control plane/worker node 1","Update control plane/worker node 2","Update control plane/worker node 3","Update pure worker node 1","Update pure worker node 2"]}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "add a task with a step for each of the 5 nodes"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending == nil {
		t.Fatalf("pending = nil, reply = %q", outcome.Reply)
	}
	if tasks, err := st.ListTasks(ctx, store.ListTasksParams{}); err != nil || len(tasks) != 0 {
		t.Fatalf("tasks before confirmation = %#v, err = %v", tasks, err)
	}

	if _, err := server.ConfirmChatAction(ctx, *outcome.Pending); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasks(ctx, store.ListTasksParams{})
	if err != nil {
		t.Fatal(err)
	}
	var parent store.Task
	for _, task := range tasks {
		if task.Title == "Update homelab TalosOS version" && task.Kind == "task" {
			parent = task
			break
		}
	}
	if parent.ID == 0 {
		t.Fatalf("parent task missing: %#v", tasks)
	}
	steps, err := st.ListSubtasks(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 5 {
		t.Fatalf("steps = %d, want 5", len(steps))
	}
	for _, step := range steps {
		if step.Kind != "step" || step.ParentTaskID == nil || *step.ParentTaskID != parent.ID {
			t.Errorf("step %q: kind=%q parent=%v", step.Title, step.Kind, step.ParentTaskID)
		}
	}
}

func TestChatCreateArcNestsTasksAndSteps(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	provider := &fakeAI{reply: `I will create the arc.
ACTION: {"type":"create_arc","title":"Migrate the homelab","tasks":[{"title":"Update TalosOS","steps":["Update control plane 1","Update worker 1"]},{"title":"Update Cilium","steps":[]}]}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "create an arc with tasks and steps"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending == nil {
		t.Fatalf("pending = nil, reply = %q", outcome.Reply)
	}
	if tasks, err := st.ListTasks(ctx, store.ListTasksParams{}); err != nil || len(tasks) != 0 {
		t.Fatalf("tasks before confirmation = %#v, err = %v", tasks, err)
	}

	if _, err := server.ConfirmChatAction(ctx, *outcome.Pending); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasks(ctx, store.ListTasksParams{})
	if err != nil {
		t.Fatal(err)
	}
	var arc store.Task
	for _, task := range tasks {
		if task.Title == "Migrate the homelab" {
			arc = task
			break
		}
	}
	if arc.ID == 0 || arc.Kind != "arc" || arc.ParentTaskID != nil {
		t.Fatalf("arc = %#v", arc)
	}
	children, err := st.ListSubtasks(ctx, arc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 {
		t.Fatalf("tasks = %d, want 2", len(children))
	}
	steps, err := st.ListSubtasks(ctx, children[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(steps))
	}
	for _, step := range steps {
		if step.Kind != "step" || step.ParentTaskID == nil || *step.ParentTaskID != children[0].ID {
			t.Errorf("step %q: kind=%q parent=%v", step.Title, step.Kind, step.ParentTaskID)
		}
	}
}

func TestChatCreatesOneTaskPerActionLine(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	provider := &fakeAI{reply: `I will create the tasks.
ACTION: {"type":"create_task","title":"Update node 1","due_date":""}
ACTION: {"type":"create_task","title":"Update node 2","due_date":""}
ACTION: {"type":"create_task","title":"Update node 3","due_date":""}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "create one task per node"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending == nil || len(outcome.Pending.Actions) != 3 {
		t.Fatalf("pending = %#v, reply = %q", outcome.Pending, outcome.Reply)
	}
	if strings.Contains(outcome.Reply, "ACTION:") {
		t.Errorf("reply leaked action lines: %q", outcome.Reply)
	}
	if tasks, err := st.ListTasks(ctx, store.ListTasksParams{}); err != nil || len(tasks) != 0 {
		t.Fatalf("tasks before confirmation = %#v, err = %v", tasks, err)
	}

	if _, err := server.ConfirmChatAction(ctx, *outcome.Pending); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasks(ctx, store.ListTasksParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Fatalf("tasks = %#v", tasks)
	}
	for _, title := range []string{"Update node 1", "Update node 2", "Update node 3"} {
		found := false
		for _, task := range tasks {
			if task.Title == title {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing task %q", title)
		}
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

func TestChatCreateReminderIsPendingUntilConfirmed(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	provider := &fakeAI{reply: `ACTION: {"type":"create_reminder","title":"Call the dentist","remind_at":"2099-01-01T10:00:00Z"}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "remind me to call the dentist"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending == nil {
		t.Fatalf("pending = nil, reply = %q", outcome.Reply)
	}
	if reminders, err := st.ListReminders(ctx, nil); err != nil || len(reminders) != 0 {
		t.Fatalf("reminders before confirmation = %#v, err = %v", reminders, err)
	}

	if _, err := server.ConfirmChatAction(ctx, *outcome.Pending); err != nil {
		t.Fatal(err)
	}
	reminders, err := st.ListReminders(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 1 || reminders[0].Title != "Call the dentist" {
		t.Fatalf("reminders = %#v", reminders)
	}
}

func TestChatConfirmsMoveAndCreateTogether(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	todoID := columnID(t, st, "To Do")
	doneID := columnID(t, st, "Done")
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: todoID, Title: "Clayface", Priority: "medium", Position: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	provider := &fakeAI{reply: `ACTION: {"type":"move_task","task":"@` + task.Ref + `","column":"Done"}
ACTION: {"type":"create_task","title":"Watch the next episode"}`}
	server := &Server{Store: st, AI: provider}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "move Clayface to done and add a task to watch the next episode"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending == nil || len(outcome.Pending.Actions) != 2 {
		t.Fatalf("pending = %#v, reply = %q", outcome.Pending, outcome.Reply)
	}
	if !strings.Contains(outcome.Reply, "Confirm these changes") {
		t.Fatalf("reply = %q", outcome.Reply)
	}

	if _, err := server.ConfirmChatAction(ctx, *outcome.Pending); err != nil {
		t.Fatal(err)
	}
	moved, err := st.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.ColumnID != doneID {
		t.Fatalf("column = %d, want %d", moved.ColumnID, doneID)
	}
	tasks, err := st.ListTasks(ctx, store.ListTasksParams{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, task := range tasks {
		if task.Title == "Watch the next episode" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created task missing: %#v", tasks)
	}
}
