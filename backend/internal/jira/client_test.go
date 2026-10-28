package jira

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchIssuesFollowsNextPageToken(t *testing.T) {
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("nextPageToken")
		pages = append(pages, token)
		w.Header().Set("Content-Type", "application/json")
		if token == "" {
			fmt.Fprint(w, `{"issues":[{"key":"DEV-1","fields":{"summary":"One","status":{"name":"To Do"}}}],"nextPageToken":"page-2"}`)
			return
		}
		fmt.Fprint(w, `{"issues":[{"key":"DEV-2","fields":{"summary":"Two","status":{"name":"Done"}}}],"isLast":true}`)
	}))
	defer srv.Close()

	issues, err := New(srv.URL, "email", "token").SearchIssues("project = DEV", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 || issues[1].Key != "DEV-2" {
		t.Fatalf("issues = %#v", issues)
	}
	if len(pages) != 2 || pages[0] != "" || pages[1] != "page-2" {
		t.Fatalf("page tokens = %#v", pages)
	}
}
