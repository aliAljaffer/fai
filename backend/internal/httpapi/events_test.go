package httpapi

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestEventsStreamsSuccessfulMutations(t *testing.T) {
	s := &Server{}
	mux := chi.NewRouter()
	mux.Use(s.broadcastMutations)
	mux.Get("/events", s.events)
	mux.Post("/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}

	reader := bufio.NewReader(resp.Body)
	if line, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, ": connected") {
		t.Fatalf("greeting = %q, %v", line, err)
	}
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatal(err)
	}

	if resp, err := ts.Client().Post(ts.URL+"/tasks", "application/json", nil); err != nil {
		t.Fatal(err)
	} else {
		resp.Body.Close()
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, `"topic":"tasks"`) {
		t.Fatalf("event = %q", line)
	}
}
