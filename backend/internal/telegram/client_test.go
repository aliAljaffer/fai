package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetUpdatesSendsOffsetAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botsecret/getUpdates" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("offset") != "5" || r.URL.Query().Get("timeout") != "50" {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":6,"message":{"message_id":1,"from":{"id":7},"chat":{"id":70,"type":"private"},"text":"hi"}}]}`))
	}))
	defer server.Close()

	client := NewWithBaseURL("secret", server.URL)
	updates, err := client.GetUpdates(context.Background(), 5, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 6 {
		t.Fatalf("updates = %#v", updates)
	}
	if updates[0].Message == nil || updates[0].Message.Text != "hi" {
		t.Fatalf("message = %#v", updates[0].Message)
	}
}

func TestSendMessageSendsInlineKeyboard(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botsecret/sendMessage" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{
		{Text: "Confirm", CallbackData: "confirm:abc"},
		{Text: "Cancel", CallbackData: "cancel:abc"},
	}}}
	client := NewWithBaseURL("secret", server.URL)
	if err := client.SendMessage(context.Background(), 70, "Move it?", markup); err != nil {
		t.Fatal(err)
	}
	if payload["chat_id"] != float64(70) || payload["text"] != "Move it?" {
		t.Fatalf("payload = %#v", payload)
	}
	reply, _ := payload["reply_markup"].(map[string]any)
	rows, _ := reply["inline_keyboard"].([]any)
	if len(rows) != 1 {
		t.Fatalf("keyboard = %#v", reply)
	}
	row, _ := rows[0].([]any)
	if len(row) != 2 {
		t.Fatalf("row = %#v", row)
	}
}

func TestClientErrorsHideToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"description":"bad token secret"}`))
	}))
	defer server.Close()

	client := NewWithBaseURL("secret", server.URL)
	_, err := client.GetUpdates(context.Background(), 0, 0)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error exposes token: %v", err)
	}
}
