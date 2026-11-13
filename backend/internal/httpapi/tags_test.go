package httpapi

import (
	"net/http"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func TestTagsAreListedPerWorkspace(t *testing.T) {
	_, server := newWorkspaceServer(t)

	res := postJSON(t, server.URL+"/api/workspaces", `{"name":"Work","tag":"WRK"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create workspace status = %d", res.StatusCode)
	}
	second := decodeInto[store.Workspace](t, res)

	res = postJSON(t, server.URL+"/api/tags", `{"name":"bug"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("create tag without a workspace status = %d", res.StatusCode)
	}
	res.Body.Close()

	for _, body := range []string{
		`{"name":"bug","workspace_id":1}`,
		`{"name":"bug","workspace_id":` + itoa(second.ID) + `}`,
		`{"name":"release","workspace_id":` + itoa(second.ID) + `}`,
	} {
		res := postJSON(t, server.URL+"/api/tags", body)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("create tag %s status = %d", body, res.StatusCode)
		}
		res.Body.Close()
	}

	cases := map[string]int{
		"/api/tags":                3,
		"/api/tags?workspace_id=1": 1,
		"/api/tags?workspace_id=" + itoa(second.ID): 2,
	}
	for path, want := range cases {
		got := decodeInto[[]store.Tag](t, httpGet(t, server.URL+path))
		if len(got) != want {
			t.Fatalf("%s returned %d tags, want %d", path, len(got), want)
		}
	}
}
