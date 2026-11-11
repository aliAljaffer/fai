package httpapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const chatReminderLayout = "Mon 2006-01-02 15:04"

func (s *Server) prepareChatCreateReminder(_ context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	action.Title = strings.TrimSpace(action.Title)
	if action.Title == "" {
		return chatAction{}, "", false, fmt.Errorf("the action has no title")
	}
	remindAt, err := time.Parse(time.RFC3339, action.RemindAt)
	if err != nil {
		return chatAction{}, "", false, fmt.Errorf("invalid remind_at")
	}
	change, err := decodeRepeat(action.Repeat)
	if err != nil {
		return chatAction{}, "", false, err
	}
	action.RemindAt = remindAt.Format(time.RFC3339)
	text := "Create reminder \"" + action.Title + "\" at " + remindAt.Format(chatReminderLayout)
	if change.rule != nil {
		text += ", repeats " + change.rule.Describe()
	}
	return action, text + "?", true, nil
}

func (s *Server) confirmChatCreateReminder(ctx context.Context, action chatAction) (string, error) {
	title := strings.TrimSpace(action.Title)
	if title == "" {
		return "", fmt.Errorf("the action has no title")
	}
	remindAt, err := time.Parse(time.RFC3339, action.RemindAt)
	if err != nil {
		return "", fmt.Errorf("invalid remind_at")
	}
	change, err := decodeRepeat(action.Repeat)
	if err != nil {
		return "", err
	}
	reminder, err := s.Store.CreateReminder(ctx, title, remindAt, nil, 0, reminderRepeat(change, remindAt))
	if err != nil {
		return "", err
	}
	s.broadcast("reminders")
	return "Created reminder \"" + reminder.Title + "\".", nil
}

func chatNextTime(at time.Time) string {
	return at.UTC().Format(chatReminderLayout) + " UTC"
}

func repeatChangeText(change repeatChange) string {
	if change.rule == nil {
		return "no repeat"
	}
	return "repeat " + change.rule.Describe()
}

func (s *Server) chatReminder(ctx context.Context, id int64) (store.Reminder, bool, error) {
	if id == 0 {
		return store.Reminder{}, false, nil
	}
	reminders, err := s.Store.ListReminders(ctx, nil)
	if err != nil {
		return store.Reminder{}, false, err
	}
	for _, reminder := range reminders {
		if reminder.ID == id {
			return reminder, true, nil
		}
	}
	return store.Reminder{}, false, nil
}

const chatReminderMissing = "I could not find that reminder. Use a reminder id from the reminders list."

func (s *Server) confirmedChatReminder(ctx context.Context, id int64) (store.Reminder, error) {
	reminder, found, err := s.chatReminder(ctx, id)
	if err != nil {
		return store.Reminder{}, err
	}
	if !found {
		return store.Reminder{}, fmt.Errorf("the reminder no longer exists")
	}
	return reminder, nil
}

func chatReminderChange(action chatAction, current store.Reminder) (string, time.Time, *store.ReminderRepeat, []string, error) {
	title, remindAt := current.Title, current.RemindAt
	var changes []string
	if next := strings.TrimSpace(action.Title); next != "" && next != current.Title {
		title = next
		changes = append(changes, "title to \""+next+"\"")
	}
	if action.RemindAt != "" {
		parsed, err := time.Parse(time.RFC3339, action.RemindAt)
		if err != nil {
			return "", time.Time{}, nil, nil, fmt.Errorf("invalid remind_at")
		}
		remindAt = parsed
		changes = append(changes, "time to "+parsed.Format(chatReminderLayout))
	}
	change, err := decodeRepeat(action.Repeat)
	if err != nil {
		return "", time.Time{}, nil, nil, err
	}
	if change.set {
		changes = append(changes, repeatChangeText(change))
	}
	return title, remindAt, reminderRepeatForUpdate(current, remindAt, change, true), changes, nil
}

func (s *Server) prepareChatUpdateReminder(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	reminder, found, err := s.chatReminder(ctx, action.ReminderID)
	if err != nil || !found {
		return chatAction{}, chatReminderMissing, false, err
	}
	_, _, _, changes, err := chatReminderChange(action, reminder)
	if err != nil {
		return chatAction{}, "", false, err
	}
	if len(changes) == 0 {
		return chatAction{}, "What should I change on the reminder \"" + reminder.Title + "\"?", false, nil
	}
	return action, "Change reminder \"" + reminder.Title + "\": " + strings.Join(changes, ", ") + "?", true, nil
}

func (s *Server) confirmChatUpdateReminder(ctx context.Context, action chatAction) (string, error) {
	reminder, err := s.confirmedChatReminder(ctx, action.ReminderID)
	if err != nil {
		return "", err
	}
	title, remindAt, repeat, _, err := chatReminderChange(action, reminder)
	if err != nil {
		return "", err
	}
	updated, err := s.Store.UpdateReminder(ctx, reminder.ID, title, remindAt, reminder.TaskID, repeat)
	if err != nil {
		return "", err
	}
	s.broadcast("reminders")
	return "Updated reminder \"" + updated.Title + "\".", nil
}

func (s *Server) prepareChatCompleteReminder(ctx context.Context, action chatAction, now time.Time) (chatAction, string, bool, error) {
	reminder, found, err := s.chatReminder(ctx, action.ReminderID)
	if err != nil || !found {
		return chatAction{}, chatReminderMissing, false, err
	}
	if reminder.CompletedAt != nil {
		return chatAction{}, "The reminder \"" + reminder.Title + "\" is already done.", false, nil
	}
	if state := reminder.RepeatState(); state != nil {
		next := nextReminderTime(*state, reminder.RemindAt, now)
		return action, "Mark reminder \"" + reminder.Title + "\" done? It repeats " + state.Rule.Describe() + ", so it moves to " + chatNextTime(next) + ".", true, nil
	}
	return action, "Mark reminder \"" + reminder.Title + "\" done?", true, nil
}

func (s *Server) confirmChatCompleteReminder(ctx context.Context, action chatAction) (string, error) {
	reminder, err := s.confirmedChatReminder(ctx, action.ReminderID)
	if err != nil {
		return "", err
	}
	updated, err := s.completeReminderItem(ctx, reminder, time.Now())
	if err != nil {
		return "", err
	}
	s.broadcast("reminders")
	if updated.Repeat != nil {
		return "Marked reminder \"" + reminder.Title + "\" done. It repeats, so it moved to " + chatNextTime(updated.RemindAt) + ".", nil
	}
	return "Marked reminder \"" + reminder.Title + "\" done.", nil
}

func (s *Server) prepareChatDeleteReminder(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	reminder, found, err := s.chatReminder(ctx, action.ReminderID)
	if err != nil || !found {
		return chatAction{}, chatReminderMissing, false, err
	}
	return action, "Delete reminder \"" + reminder.Title + "\"? This cannot be undone.", true, nil
}

func (s *Server) confirmChatDeleteReminder(ctx context.Context, action chatAction) (string, error) {
	reminder, err := s.confirmedChatReminder(ctx, action.ReminderID)
	if err != nil {
		return "", err
	}
	if err := s.Store.DeleteReminder(ctx, reminder.ID); err != nil {
		return "", err
	}
	s.broadcast("reminders")
	return "Deleted reminder \"" + reminder.Title + "\".", nil
}

func (s *Server) chatNotesByTitle(ctx context.Context, title string) ([]store.Note, error) {
	notes, err := s.Store.ListNotes(ctx, nil)
	if err != nil {
		return nil, err
	}
	var matches []store.Note
	for _, note := range notes {
		if strings.EqualFold(strings.TrimSpace(note.Title), title) {
			matches = append(matches, note)
		}
	}
	return matches, nil
}

func (s *Server) prepareChatCreateNote(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	action.Title = strings.TrimSpace(action.Title)
	action.Body = strings.TrimSpace(action.Body)
	if action.Title == "" {
		return chatAction{}, "What should the note be called?", false, nil
	}
	existing, err := s.chatNotesByTitle(ctx, action.Title)
	if err != nil {
		return chatAction{}, "", false, err
	}
	if len(existing) > 0 {
		return chatAction{}, "A note named \"" + action.Title + "\" already exists. Should I add to it instead?", false, nil
	}
	return action, "Create note \"" + action.Title + "\"?", true, nil
}

func (s *Server) confirmChatCreateNote(ctx context.Context, action chatAction) (string, error) {
	title := strings.TrimSpace(action.Title)
	if title == "" {
		return "", fmt.Errorf("the note has no title")
	}
	note, err := s.Store.CreateNote(ctx, title, strings.TrimSpace(action.Body), nil, 0)
	if err != nil {
		return "", err
	}
	s.broadcast("notes")
	return "Created note \"" + note.Title + "\".", nil
}

func (s *Server) prepareChatAppendNote(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	action.Body = strings.TrimSpace(action.Body)
	matches, err := s.chatNotesByTitle(ctx, strings.TrimSpace(action.Note))
	if err != nil {
		return chatAction{}, "", false, err
	}
	if len(matches) != 1 {
		return chatAction{}, "I could not find one note with that title. Copy the title from the notes list.", false, nil
	}
	note := matches[0]
	if action.Body == "" {
		return chatAction{}, "What should I add to the note \"" + note.Title + "\"?", false, nil
	}
	action.NoteID = note.ID
	action.Note = note.Title
	return action, "Add to note \"" + note.Title + "\": " + quoteSnippet(action.Body) + "?", true, nil
}

func (s *Server) confirmChatAppendNote(ctx context.Context, action chatAction) (string, error) {
	addition := strings.TrimSpace(action.Body)
	if addition == "" {
		return "", fmt.Errorf("there is nothing to add")
	}
	note, err := s.Store.GetNote(ctx, action.NoteID)
	if err != nil || action.NoteID == 0 {
		return "", fmt.Errorf("the note no longer exists")
	}
	body := addition
	if existing := strings.TrimRight(note.BodyMd, "\n"); existing != "" {
		body = existing + "\n\n" + addition
	}
	if _, err := s.Store.UpdateNote(ctx, note.ID, note.Title, body, note.TaskID); err != nil {
		return "", err
	}
	s.broadcast("notes")
	return "Added to note \"" + note.Title + "\".", nil
}
