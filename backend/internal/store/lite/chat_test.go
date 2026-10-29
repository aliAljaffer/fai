package lite

import (
	"context"
	"testing"
)

func TestChatMessagesPersistAndBacklink(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	board := mustBoard(t, st, "Personal", 0)
	columnID := columnOf(t, st, board.ID)
	task := mustTask(t, st, board.ID, columnID, "Ship release", "")

	if _, err := st.CreateChatMessage(ctx, board.WorkspaceID, "user", "what about@"+task.Ref+"?"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateChatMessage(ctx, board.WorkspaceID, "assistant", "It is open."); err != nil {
		t.Fatal(err)
	}

	messages, err := st.ListChatMessages(ctx, board.WorkspaceID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("messages = %#v", messages)
	}

	links, err := st.ListTaskReferences(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, link := range links {
		if link.Kind == "chat" {
			found = true
		}
	}
	if !found {
		t.Errorf("chat backlink missing: %#v", links)
	}
}
