package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

func TestTaskCommentCRUD(t *testing.T) {
	st, server := newWorkspaceServer(t)
	task, err := st.CreateTask(context.Background(), store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Task", Priority: "medium", Position: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	res := postJSON(t, server.URL+"/api/tasks/"+itoa(task.ID)+"/comments", `{"body_md":"Hello"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", res.StatusCode)
	}
	created := decodeInto[store.TaskComment](t, res)
	if created.BodyMd != "Hello" || created.TaskID != task.ID {
		t.Fatalf("created = %#v", created)
	}

	res = postJSON(t, server.URL+"/api/tasks/"+itoa(task.ID)+"/comments", `{"body_md":" "}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("empty comment status = %d", res.StatusCode)
	}
	res.Body.Close()

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/tasks/"+itoa(task.ID)+"/comments/"+itoa(created.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", res.StatusCode)
	}
	res.Body.Close()

	comments, err := st.ListTaskComments(context.Background(), task.ID)
	if err != nil || len(comments) != 0 {
		t.Fatalf("comments after delete = %#v, err = %v", comments, err)
	}
}
