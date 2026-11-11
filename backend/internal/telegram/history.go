package telegram

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/alialjaffer/fai/backend/internal/ai"
)

type HistoryStore interface {
	Load() (map[int64][]ai.ChatMessage, error)
	Save(map[int64][]ai.ChatMessage) error
}

type FileHistoryStore struct {
	Path string
}

func (s FileHistoryStore) Load() (map[int64][]ai.ChatMessage, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw map[string][]ai.ChatMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("telegram: invalid history file: %w", err)
	}
	histories := make(map[int64][]ai.ChatMessage, len(raw))
	for key, messages := range raw {
		chatID, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("telegram: invalid history chat id %q: %w", key, err)
		}
		histories[chatID] = trimHistory(messages)
	}
	return histories, nil
}

func (s FileHistoryStore) Save(histories map[int64][]ai.ChatMessage) error {
	raw := make(map[string][]ai.ChatMessage, len(histories))
	for chatID, messages := range histories {
		raw[strconv.FormatInt(chatID, 10)] = trimHistory(messages)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return writeFileAtomic(s.Path, ".telegram-history-*", data)
}

func trimHistory(messages []ai.ChatMessage) []ai.ChatMessage {
	if len(messages) > historyLimit {
		return messages[len(messages)-historyLimit:]
	}
	return messages
}
