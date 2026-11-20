package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNotifierSignsAndDelivers(t *testing.T) {
	received := make(chan Event, 1)
	var signatureOK bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		timestamp := r.Header.Get("X-Webhook-Timestamp")
		mac := hmac.New(sha256.New, []byte("test-secret"))
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
		mac.Write(body)
		signatureOK = hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(r.Header.Get("X-Webhook-Signature-V2")))
		if r.Header.Get("X-Request-ID") == "" {
			t.Error("missing X-Request-ID")
		}
		var event Event
		if err := json.Unmarshal(body, &event); err != nil {
			t.Error(err)
		}
		received <- event
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	notifier := New(server.URL, "test-secret")
	if notifier == nil {
		t.Fatal("notifier is nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go notifier.Run(ctx)

	notifier.Publish(Event{Event: "task.created", Task: Task{Ref: "WORK-1", Title: "hello"}})

	select {
	case event := <-received:
		if event.Event != "task.created" || event.Task.Ref != "WORK-1" {
			t.Fatalf("event = %+v", event)
		}
		if event.ID == "" || event.At.IsZero() {
			t.Fatalf("event id or timestamp missing: %+v", event)
		}
		if !signatureOK {
			t.Fatal("signature did not verify")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no delivery")
	}
}

func TestNewRequiresURLAndSecret(t *testing.T) {
	if New("", "secret") != nil {
		t.Fatal("notifier without a URL should be nil")
	}
	if New("http://example.com/hook", "") != nil {
		t.Fatal("notifier without a secret should be nil")
	}
}
