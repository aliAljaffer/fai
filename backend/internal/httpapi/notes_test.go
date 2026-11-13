package httpapi

import (
	"net/http"
	"testing"
)

func TestCountNotesByWorkspace(t *testing.T) {
	_, server := newWorkspaceServer(t)

	for _, title := range []string{"One", "Two"} {
		res := postJSON(t, server.URL+"/api/notes", `{"title":"`+title+`","body_md":"body","workspace_id":1}`)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("create note status = %d", res.StatusCode)
		}
		res.Body.Close()
	}

	cases := map[string]int64{
		"/api/notes/count":                  2,
		"/api/notes/count?workspace_id=1":   2,
		"/api/notes/count?workspace_id=999": 0,
	}
	for path, want := range cases {
		got := decodeInto[map[string]int64](t, httpGet(t, server.URL+path))
		if got["count"] != want {
			t.Fatalf("%s count = %d, want %d", path, got["count"], want)
		}
	}
}
