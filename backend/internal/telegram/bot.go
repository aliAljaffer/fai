package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/httpapi"
)

const (
	historyLimit     = 10
	maxVoiceSeconds  = 90
	pendingTTL       = 5 * time.Minute
	pollTimeout      = 50
	confirmPrefix    = "confirm:"
	cancelPrefix     = "cancel:"
	maxMessageLength = 4000
)

type ChatRunner interface {
	RunChat(ctx context.Context, messages []ai.ChatMessage, tzOffset *int) (httpapi.ChatOutcome, error)
	ConfirmChatAction(ctx context.Context, pending httpapi.PendingChatAction) (string, error)
	Transcribe(ctx context.Context, audio []byte, format string) (string, error)
}

type pendingEntry struct {
	action  httpapi.PendingChatAction
	expires time.Time
}

type Bot struct {
	client        *Client
	runner        ChatRunner
	allowedUserID int64
	allowedChatID int64
	now           func() time.Time
	offsets       OffsetStore

	mu      sync.Mutex
	history map[int64][]ai.ChatMessage
	pending map[string]pendingEntry
}

func NewBot(client *Client, allowedUserID, allowedChatID int64, runner ChatRunner) *Bot {
	if client == nil || runner == nil || allowedUserID == 0 || allowedChatID == 0 {
		return nil
	}
	return &Bot{
		client:        client,
		runner:        runner,
		allowedUserID: allowedUserID,
		allowedChatID: allowedChatID,
		now:           time.Now,
		history:       make(map[int64][]ai.ChatMessage),
		pending:       make(map[string]pendingEntry),
	}
}

func (b *Bot) WithOffsetStore(store OffsetStore) *Bot {
	if b != nil {
		b.offsets = store
	}
	return b
}

func (b *Bot) Run(ctx context.Context) {
	if b == nil {
		return
	}
	var offset int64
	if b.offsets != nil {
		saved, err := b.offsets.Load()
		if err != nil {
			log.Printf("telegram: load offset: %v", err)
		} else {
			offset = saved
		}
	}
	for {
		updates, err := b.client.GetUpdates(ctx, offset, pollTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("telegram: poll: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}
		for _, update := range updates {
			b.handleUpdate(ctx, update)
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
				if b.offsets != nil {
					if err := b.offsets.Save(offset); err != nil {
						log.Printf("telegram: save offset: %v", err)
					}
				}
			}
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update Update) {
	if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery)
		return
	}
	if update.Message == nil {
		return
	}
	b.handleMessage(ctx, update.Message)
}

func (b *Bot) handleMessage(ctx context.Context, message *Message) {
	if !b.allowed(message.Chat, message.From) {
		return
	}
	if message.Voice != nil {
		b.handleVoice(ctx, message)
		return
	}
	text := strings.TrimSpace(message.Text)
	if text == "" {
		return
	}
	if text == "/start" || text == "/help" {
		b.sendText(ctx, message.Chat.ID, "Send text or a voice note. I can create tasks and reminders, find tasks, and move a task to another column. I will ask before moving a task.")
		return
	}
	b.respond(ctx, message.Chat.ID, text, "")
}

func (b *Bot) allowed(chat Chat, from *User) bool {
	if chat.Type != "private" || from == nil || from.IsBot {
		return false
	}
	return from.ID == b.allowedUserID && chat.ID == b.allowedChatID
}

func (b *Bot) handleVoice(ctx context.Context, message *Message) {
	chatID := message.Chat.ID
	if message.Voice.Duration > maxVoiceSeconds {
		b.sendText(ctx, chatID, fmt.Sprintf("Voice notes are limited to %d seconds.", maxVoiceSeconds))
		return
	}
	file, err := b.client.GetFile(ctx, message.Voice.FileID)
	if err != nil || file.FilePath == "" {
		log.Printf("telegram: get file: %v", err)
		b.sendText(ctx, chatID, "Could not download that voice note.")
		return
	}
	audio, err := b.client.DownloadFile(ctx, file.FilePath)
	if err != nil {
		log.Printf("telegram: download voice: %v", err)
		b.sendText(ctx, chatID, "Could not download that voice note.")
		return
	}
	format := audioFormat(message.Voice.MimeType)
	transcript, err := b.runner.Transcribe(ctx, audio, format)
	if err != nil && format == "ogg" {
		converted, convErr := transcodeToMP3(ctx, audio)
		switch {
		case convErr != nil:
			log.Printf("telegram: transcode: %v", convErr)
		default:
			transcript, err = b.runner.Transcribe(ctx, converted, "mp3")
		}
	}
	if err != nil || strings.TrimSpace(transcript) == "" {
		if err != nil {
			log.Printf("telegram: transcribe: %v", err)
		}
		b.sendText(ctx, chatID, "Could not transcribe that voice note. Send it as text.")
		return
	}
	transcript = strings.TrimSpace(transcript)
	b.respond(ctx, chatID, transcript, "Heard: "+transcript)
}

func audioFormat(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/wav", "audio/x-wav":
		return "wav"
	case "audio/mp4", "audio/m4a":
		return "m4a"
	default:
		return "ogg"
	}
}

func (b *Bot) respond(ctx context.Context, chatID int64, text, prefix string) {
	b.mu.Lock()
	history := append([]ai.ChatMessage(nil), b.history[chatID]...)
	b.mu.Unlock()
	history = append(history, ai.ChatMessage{Role: "user", Content: text})
	if len(history) > historyLimit {
		history = history[len(history)-historyLimit:]
	}

	outcome, err := b.runner.RunChat(ctx, history, nil)
	if err != nil {
		log.Printf("telegram: chat: %v", err)
		b.sendText(ctx, chatID, "Could not answer that right now.")
		return
	}
	reply := strings.TrimSpace(outcome.Reply)
	if prefix != "" {
		reply = prefix + "\n\n" + reply
	}
	b.remember(chatID, ai.ChatMessage{Role: "user", Content: text}, ai.ChatMessage{Role: "assistant", Content: reply})
	if outcome.Pending != nil {
		b.sendPending(ctx, chatID, reply, *outcome.Pending)
		return
	}
	b.sendText(ctx, chatID, reply)
}

func (b *Bot) remember(chatID int64, messages ...ai.ChatMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	history := append(b.history[chatID], messages...)
	if len(history) > historyLimit {
		history = history[len(history)-historyLimit:]
	}
	b.history[chatID] = history
}

func (b *Bot) sendText(ctx context.Context, chatID int64, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	if err := b.client.SendMessage(ctx, chatID, truncateRunes(text, maxMessageLength), nil); err != nil {
		log.Printf("telegram: send: %v", err)
	}
}

func (b *Bot) sendPending(ctx context.Context, chatID int64, text string, action httpapi.PendingChatAction) {
	token, err := randomToken()
	if err != nil {
		b.sendText(ctx, chatID, text+"\n\nCould not create confirmation buttons.")
		return
	}
	b.mu.Lock()
	b.prunePendingLocked()
	b.pending[token] = pendingEntry{action: action, expires: b.now().Add(pendingTTL)}
	b.mu.Unlock()

	markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{
		{Text: "Confirm", CallbackData: confirmPrefix + token},
		{Text: "Cancel", CallbackData: cancelPrefix + token},
	}}}
	if err := b.client.SendMessage(ctx, chatID, truncateRunes(text, maxMessageLength), markup); err != nil {
		log.Printf("telegram: send confirmation: %v", err)
		b.mu.Lock()
		delete(b.pending, token)
		b.mu.Unlock()
	}
}

func (b *Bot) handleCallback(ctx context.Context, callback *CallbackQuery) {
	if callback.Message == nil || !b.allowed(callback.Message.Chat, callback.From) {
		return
	}
	chatID := callback.Message.Chat.ID

	switch {
	case strings.HasPrefix(callback.Data, confirmPrefix):
		entry, ok := b.takePending(strings.TrimPrefix(callback.Data, confirmPrefix))
		if !ok {
			_ = b.client.AnswerCallbackQuery(ctx, callback.ID, "This action expired.")
			b.sendText(ctx, chatID, "That action expired. Ask again.")
			return
		}
		_ = b.client.AnswerCallbackQuery(ctx, callback.ID, "Confirmed")
		reply, err := b.runner.ConfirmChatAction(ctx, entry.action)
		if err != nil {
			log.Printf("telegram: confirm: %v", err)
			b.sendText(ctx, chatID, "Could not do that: "+err.Error())
			return
		}
		b.sendText(ctx, chatID, reply)
	case strings.HasPrefix(callback.Data, cancelPrefix):
		if _, ok := b.takePending(strings.TrimPrefix(callback.Data, cancelPrefix)); !ok {
			_ = b.client.AnswerCallbackQuery(ctx, callback.ID, "This action expired.")
			b.sendText(ctx, chatID, "That action expired.")
			return
		}
		_ = b.client.AnswerCallbackQuery(ctx, callback.ID, "Cancelled")
		b.sendText(ctx, chatID, "Cancelled.")
	}
}

func (b *Bot) takePending(token string) (pendingEntry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.prunePendingLocked()
	entry, ok := b.pending[token]
	if !ok {
		return pendingEntry{}, false
	}
	delete(b.pending, token)
	return entry, true
}

func (b *Bot) prunePendingLocked() {
	now := b.now()
	for token, entry := range b.pending {
		if now.After(entry.expires) {
			delete(b.pending, token)
		}
	}
}

func randomToken() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}
