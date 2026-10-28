package jira

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCommentsPaginates(t *testing.T) {
	var starts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("startAt")
		starts = append(starts, start)
		w.Header().Set("Content-Type", "application/json")
		if start == "0" {
			fmt.Fprint(w, `{"comments":[{"id":"1","author":{"accountId":"me","displayName":"Me"},"body":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"One"}]}]}}],"startAt":0,"total":2}`)
			return
		}
		fmt.Fprint(w, `{"comments":[{"id":"2","author":{"accountId":"me","displayName":"Me"},"body":"Two"}],"startAt":1,"total":2}`)
	}))
	defer srv.Close()

	comments, err := New(srv.URL, "email", "token").GetComments("DEV-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 || comments[0].Body != "One" || comments[1].Body != "Two" {
		t.Fatalf("comments = %#v", comments)
	}
	if len(starts) != 2 || starts[0] != "0" || starts[1] != "1" {
		t.Fatalf("startAt values = %#v", starts)
	}
}

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
