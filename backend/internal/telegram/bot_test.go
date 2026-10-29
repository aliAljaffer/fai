package telegram

import (
	"testing"
	"time"
)

func TestAllowedRequiresPrivateMatchingUserAndChat(t *testing.T) {
	bot := &Bot{allowedUserID: 7, allowedChatID: 70}
	if !bot.allowed(Chat{ID: 70, Type: "private"}, &User{ID: 7}) {
		t.Fatal("expected the owner to be allowed")
	}
	cases := []struct {
		name string
		chat Chat
		from *User
	}{
		{"wrong user", Chat{ID: 70, Type: "private"}, &User{ID: 8}},
		{"wrong chat", Chat{ID: 71, Type: "private"}, &User{ID: 7}},
		{"group chat", Chat{ID: 70, Type: "group"}, &User{ID: 7}},
		{"bot sender", Chat{ID: 70, Type: "private"}, &User{ID: 7, IsBot: true}},
		{"missing sender", Chat{ID: 70, Type: "private"}, nil},
	}
	for _, tc := range cases {
		if bot.allowed(tc.chat, tc.from) {
			t.Errorf("%s: expected to be dropped", tc.name)
		}
	}
}

func TestTakePendingRemovesAndExpires(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	bot := &Bot{
		now: func() time.Time { return now },
		pending: map[string]pendingEntry{
			"live":    {expires: now.Add(time.Minute)},
			"expired": {expires: now.Add(-time.Minute)},
		},
	}
	if _, ok := bot.takePending("expired"); ok {
		t.Fatal("expired entry was returned")
	}
	if _, ok := bot.takePending("live"); !ok {
		t.Fatal("live entry was not returned")
	}
	if _, ok := bot.takePending("live"); ok {
		t.Fatal("entry was returned twice")
	}
}

func TestAudioFormat(t *testing.T) {
	cases := map[string]string{
		"audio/ogg":  "ogg",
		"audio/mpeg": "mp3",
		"audio/wav":  "wav",
		"audio/mp4":  "m4a",
		"":           "ogg",
	}
	for mime, want := range cases {
		if got := audioFormat(mime); got != want {
			t.Errorf("audioFormat(%q) = %q, want %q", mime, got, want)
		}
	}
}
