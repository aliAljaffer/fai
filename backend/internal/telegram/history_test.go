package telegram

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/ai"
)

func testBot() *Bot {
	return &Bot{history: make(map[int64][]ai.ChatMessage)}
}

func TestFileHistoryStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "history")
	store := FileHistoryStore{Path: path}
	want := map[int64][]ai.ChatMessage{
		7:  {{Role: "user", Content: "hi"}, {Role: "assistant", Content: "hello"}},
		-9: {{Role: "user", Content: "yo"}},
	}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || len(got[7]) != 2 || got[7][1].Content != "hello" || got[-9][0].Content != "yo" {
		t.Fatalf("got = %#v", got)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("stat = %v, %v", info, err)
	}
}

func TestFileHistoryStoreTrimsToLimit(t *testing.T) {
	store := FileHistoryStore{Path: filepath.Join(t.TempDir(), "history")}
	var messages []ai.ChatMessage
	for i := 0; i < historyLimit+5; i++ {
		messages = append(messages, ai.ChatMessage{Role: "user", Content: strconv.Itoa(i)})
	}
	if err := store.Save(map[int64][]ai.ChatMessage{1: messages}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got[1]) != historyLimit || got[1][0].Content != "5" {
		t.Fatalf("got %d messages starting at %q", len(got[1]), got[1][0].Content)
	}
}

func TestFileHistoryStoreMissingFile(t *testing.T) {
	got, err := FileHistoryStore{Path: filepath.Join(t.TempDir(), "none")}.Load()
	if err != nil || len(got) != 0 {
		t.Fatalf("got = %#v, err = %v", got, err)
	}
}

func TestFileHistoryStoreCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := FileHistoryStore{Path: path}
	if _, err := store.Load(); err == nil {
		t.Fatal("corrupt file loaded without an error")
	}
	bot := testBot().WithHistoryStore(store)
	if len(bot.history) != 0 {
		t.Fatalf("history = %#v", bot.history)
	}
	bot.remember(1, ai.ChatMessage{Role: "user", Content: "after"})
	if got, err := store.Load(); err != nil || got[1][0].Content != "after" {
		t.Fatalf("got = %#v, err = %v", got, err)
	}
}

func TestBotHistorySurvivesRestart(t *testing.T) {
	store := FileHistoryStore{Path: filepath.Join(t.TempDir(), "history")}
	first := testBot().WithHistoryStore(store)
	first.remember(5, ai.ChatMessage{Role: "user", Content: "one"}, ai.ChatMessage{Role: "assistant", Content: "two"})

	second := testBot().WithHistoryStore(store)
	if got := second.history[5]; len(got) != 2 || got[0].Content != "one" || got[1].Content != "two" {
		t.Fatalf("history = %#v", got)
	}
}

func TestBotRemembersWithoutStore(t *testing.T) {
	bot := testBot()
	bot.remember(5, ai.ChatMessage{Role: "user", Content: "one"})
	if len(bot.history[5]) != 1 {
		t.Fatalf("history = %#v", bot.history)
	}
}
