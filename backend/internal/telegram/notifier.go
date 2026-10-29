package telegram

import (
	"context"
	"strings"
	"time"
)

type Notifier struct {
	client *Client
	chatID int64
}

func NewNotifier(token string, chatID int64) *Notifier {
	client := New(token)
	if client == nil || chatID == 0 {
		return nil
	}
	return &Notifier{client: client, chatID: chatID}
}

func (n *Notifier) Publish(title, message string) error {
	text := strings.TrimSpace(title)
	message = strings.TrimSpace(message)
	if message != "" {
		if text != "" {
			text += "\n"
		}
		text += message
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return n.client.SendMessage(ctx, n.chatID, truncateRunes(text, 4000), nil)
}
