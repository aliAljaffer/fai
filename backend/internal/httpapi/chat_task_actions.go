package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) findChatTask(ctx context.Context, action chatAction) (string, error) {
	task, refs, err := s.resolveChatActionTask(ctx, action)
	if err != nil {
		return "", err
	}
	if task.ID == 0 {
		return chatTaskCandidates("look up", refs), nil
	}
	var out strings.Builder
	out.WriteString("@" + task.Ref + " " + task.Title)
	columns, err := s.Store.ListColumnsByBoard(ctx, task.BoardID)
	if err == nil {
		for _, column := range columns {
			if column.ID == task.ColumnID {
				out.WriteString("\nColumn: " + column.Name)
				break
			}
		}
	}
	if task.DueDate != nil {
		out.WriteString("\nDue: " + *task.DueDate)
	}
	if task.Priority != "" && task.Priority != "medium" {
		out.WriteString("\nPriority: " + task.Priority)
	}
	return out.String(), nil
}

func (s *Server) prepareChatCreate(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	title := strings.TrimSpace(action.Title)
	if title == "" {
		return chatAction{}, "", false, fmt.Errorf("the action has no title")
	}
	action.Title = title
	if _, _, err := s.defaultBoardColumn(ctx); err != nil {
		return chatAction{}, "", false, err
	}

	change, err := decodeRepeat(action.Repeat)
	if err != nil {
		return chatAction{}, "", false, err
	}

	if action.Type == "create_arc" {
		if change.rule != nil {
			return chatAction{}, "An arc cannot repeat. Which of its tasks should repeat?", false, nil
		}
		if len(action.Tasks) == 0 {
			return chatAction{}, "", false, fmt.Errorf("the arc has no tasks")
		}
		for i := range action.Tasks {
			taskTitle := strings.TrimSpace(action.Tasks[i].Title)
			if taskTitle == "" {
				return chatAction{}, "", false, fmt.Errorf("the arc has a task with no title")
			}
			action.Tasks[i].Title = taskTitle
			due, err := chatDueDate(action.Tasks[i].DueDate)
			if err != nil {
				return chatAction{}, "", false, err
			}
			action.Tasks[i].DueDate = due
		}
		return action, arcSummary(title, action.Tasks), true, nil
	}

	due, err := chatDueDate(action.DueDate)
	if err != nil {
		return chatAction{}, "", false, err
	}
	action.DueDate = due
	if _, err := taskCreateRepeat(change, "task", due); err != nil {
		return chatAction{}, chatRepeatQuestion(err), false, nil
	}
	text := "Create task \"" + title + "\""
	if due != nil {
		text += " due " + *due
	}
	if change.rule != nil {
		text += ", repeats " + change.rule.Describe()
	}
	if steps := len(nonEmptyTitles(action.Steps)); steps > 0 {
		text += fmt.Sprintf(" with %d steps", steps)
	}
	return action, text + "?", true, nil
}

func (s *Server) confirmChatCreate(ctx context.Context, action chatAction) (string, error) {
	title := strings.TrimSpace(action.Title)
	if title == "" {
		return "", fmt.Errorf("the action has no title")
	}
	boardID, columnID, err := s.defaultBoardColumn(ctx)
	if err != nil {
		return "", err
	}
	if action.Type == "create_arc" {
		arc, err := s.createChatTask(ctx, boardID, columnID, title, nil, nil, "arc", nil, nil)
		if err != nil {
			return "", err
		}
		s.broadcast("tasks")
		// Each insert reads the column's last position, so the loop stays sequential.
		for _, spec := range action.Tasks {
			taskTitle := strings.TrimSpace(spec.Title)
			if taskTitle == "" {
				continue
			}
			due, err := chatDueDate(spec.DueDate)
			if err != nil {
				return "", err
			}
			if _, err := s.createChatTask(ctx, boardID, columnID, taskTitle, due, &arc.ID, "task", spec.Steps, nil); err != nil {
				return "", err
			}
		}
		return "Created arc \"" + arc.Title + "\".", nil
	}

	due, err := chatDueDate(action.DueDate)
	if err != nil {
		return "", err
	}
	change, err := decodeRepeat(action.Repeat)
	if err != nil {
		return "", err
	}
	repeat, err := taskCreateRepeat(change, "task", due)
	if err != nil {
		return "", err
	}
	task, err := s.createChatTask(ctx, boardID, columnID, title, due, nil, "", action.Steps, repeat)
	if err != nil {
		return "", err
	}
	s.broadcast("tasks")
	return "Created task \"" + task.Title + "\".", nil
}

func arcSummary(title string, tasks []chatTaskSpec) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Create arc \"%s\" with these %d tasks?", title, len(tasks))
	for i, task := range tasks {
		fmt.Fprintf(&out, "\n%d. %s", i+1, task.Title)
		if steps := len(nonEmptyTitles(task.Steps)); steps > 0 {
			fmt.Fprintf(&out, " (%d steps)", steps)
		}
	}
	return out.String()
}

func chatDueDate(raw *string) (*string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return nil, fmt.Errorf("invalid due date %q", value)
	}
	return &value, nil
}

func (s *Server) createChatTask(ctx context.Context, boardID, columnID int64, title string, due *string, parentID *int64, kind string, steps []string, repeat *store.TaskRepeat) (store.Task, error) {
	position, err := s.appendPosition(ctx, columnID)
	if err != nil {
		return store.Task{}, err
	}
	task, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
		BoardID: boardID, ColumnID: columnID, Title: title, Priority: "medium",
		DueDate: due, Position: position, ParentTaskID: parentID, Kind: kind, CreatedByFai: true, Repeat: repeat,
	})
	if err != nil {
		return store.Task{}, err
	}
	if _, err := s.createChatChildren(ctx, task, steps); err != nil {
		return store.Task{}, err
	}
	return task, nil
}

// Each insert reads the column's last position, so the loop stays sequential.
func (s *Server) createChatChildren(ctx context.Context, parent store.Task, titles []string) (int, error) {
	kind := "step"
	if parent.Kind == "arc" {
		kind = "task"
	}
	created := 0
	for _, title := range nonEmptyTitles(titles) {
		position, err := s.appendPosition(ctx, parent.ColumnID)
		if err != nil {
			return created, err
		}
		if _, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID: parent.BoardID, ColumnID: parent.ColumnID, Title: title, Priority: "medium",
			Position: position, ParentTaskID: &parent.ID, Kind: kind, CreatedByFai: true,
		}); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

func nonEmptyTitles(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		if title := strings.TrimSpace(value); title != "" {
			out = append(out, title)
		}
	}
	return out
}

func childNoun(parent store.Task, count int) string {
	noun := "step"
	if parent.Kind == "arc" {
		noun = "task"
	}
	if count != 1 {
		noun += "s"
	}
	return fmt.Sprintf("%d %s", count, noun)
}

func (s *Server) prepareChatAddSteps(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "add steps to")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	steps := nonEmptyTitles(action.Steps)
	if len(steps) == 0 {
		return chatAction{}, "Which steps should I add to @" + task.Ref + "?", false, nil
	}
	action.Steps = steps
	return action, "Add " + childNoun(task, len(steps)) + " to " + chatTaskLabel(task) + "?", true, nil
}

func (s *Server) confirmChatAddSteps(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	created, err := s.createChatChildren(ctx, task, action.Steps)
	if created > 0 {
		s.broadcast("tasks")
	}
	if err != nil {
		return "", err
	}
	return "Added " + childNoun(task, created) + " to @" + task.Ref + ".", nil
}

func (s *Server) prepareChatMove(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "move")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	column, available, found, err := s.resolveChatColumn(ctx, task.BoardID, action.Column)
	if err != nil {
		return chatAction{}, "", false, err
	}
	if !found {
		name := strings.TrimSpace(action.Column)
		if name == "" {
			return chatAction{}, "Which column should I move @" + task.Ref + " to? Available columns: " + columnNames(available) + ".", false, nil
		}
		return chatAction{}, "I could not find a column named " + name + ". Available columns: " + columnNames(available) + ".", false, nil
	}
	action.ColumnID = column.ID
	action.Column = column.Name
	return action, "Move " + chatTaskLabel(task) + " to " + column.Name + "?", true, nil
}

func (s *Server) confirmChatMove(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	columns, err := s.Store.ListColumnsByBoard(ctx, task.BoardID)
	if err != nil {
		return "", err
	}
	for _, column := range columns {
		if column.ID == action.ColumnID {
			if err := s.moveChatTask(ctx, task, column); err != nil {
				return "", err
			}
			return "Moved @" + task.Ref + " to " + column.Name + ".", nil
		}
	}
	return "", fmt.Errorf("the column no longer exists")
}

func (s *Server) moveChatTask(ctx context.Context, task store.Task, column store.BoardColumn) error {
	position, err := s.appendPosition(ctx, column.ID)
	if err != nil {
		return err
	}
	if _, err := s.moveTaskItem(ctx, task, column.ID, position, time.Now()); err != nil {
		return err
	}
	s.broadcast("tasks")
	return nil
}

func (s *Server) resolveChatColumn(ctx context.Context, boardID int64, name string) (store.BoardColumn, []store.BoardColumn, bool, error) {
	columns, err := s.Store.ListColumnsByBoard(ctx, boardID)
	if err != nil {
		return store.BoardColumn{}, nil, false, err
	}
	name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "#"))
	if name == "" {
		return store.BoardColumn{}, columns, false, nil
	}
	matches := make([]store.BoardColumn, 0, 1)
	for _, column := range columns {
		if strings.EqualFold(strings.TrimSpace(column.Name), name) {
			matches = append(matches, column)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil, true, nil
	}
	if len(matches) > 1 {
		return store.BoardColumn{}, matches, false, nil
	}
	return store.BoardColumn{}, columns, false, nil
}

func columnNames(columns []store.BoardColumn) string {
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.Name)
	}
	return strings.Join(names, ", ")
}

func (s *Server) doneColumn(ctx context.Context, boardID int64) (store.BoardColumn, bool, error) {
	columns, err := s.Store.ListColumnsByBoard(ctx, boardID)
	if err != nil {
		return store.BoardColumn{}, false, err
	}
	for _, column := range columns {
		if column.IsDone {
			return column, true, nil
		}
	}
	return store.BoardColumn{}, false, nil
}

func (s *Server) defaultBoardColumn(ctx context.Context) (int64, int64, error) {
	boards, err := s.Store.ListBoards(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	if len(boards) == 0 {
		return 0, 0, fmt.Errorf("no board exists yet")
	}
	columns, err := s.Store.ListColumnsByBoard(ctx, boards[0].ID)
	if err != nil {
		return 0, 0, err
	}
	if len(columns) == 0 {
		return 0, 0, fmt.Errorf("board %q has no columns", boards[0].Name)
	}
	for _, column := range columns {
		if !column.IsDone {
			return boards[0].ID, column.ID, nil
		}
	}
	return boards[0].ID, columns[0].ID, nil
}

func chatTaskPatch(action chatAction, task store.Task) (store.TaskPatch, []string, error) {
	var patch store.TaskPatch
	var changes []string
	if title := strings.TrimSpace(action.Title); title != "" {
		patch.Title = &title
		changes = append(changes, "title to \""+title+"\"")
	}
	if action.Description != nil {
		description := strings.TrimSpace(*action.Description)
		patch.DescriptionMd = &description
		changes = append(changes, "description to "+quoteSnippet(description))
	}
	if action.Priority != nil {
		priority := strings.ToLower(strings.TrimSpace(*action.Priority))
		if priority != "low" && priority != "medium" && priority != "high" {
			return store.TaskPatch{}, nil, fmt.Errorf("invalid priority %q", *action.Priority)
		}
		patch.Priority = &priority
		changes = append(changes, "priority to "+priority)
	}
	if action.DueDate != nil {
		due, err := chatDueDate(action.DueDate)
		if err != nil {
			return store.TaskPatch{}, nil, err
		}
		patch.DueDate = due
		patch.DueDateSet = true
		if due == nil {
			changes = append(changes, "no due date")
		} else {
			changes = append(changes, "due date to "+*due)
		}
	}
	change, err := decodeRepeat(action.Repeat)
	if err != nil {
		return store.TaskPatch{}, nil, err
	}
	if err := applyTaskRepeat(task, &patch, change); err != nil {
		return store.TaskPatch{}, nil, chatRepeatError{err}
	}
	if change.set {
		changes = append(changes, repeatChangeText(change))
	} else if patch.RepeatSet && patch.Repeat == nil {
		changes = append(changes, "no repeat")
	}
	return patch, changes, nil
}

type chatRepeatError struct{ error }

func chatRepeatQuestion(err error) string {
	if errors.Is(err, errRepeatNeedsDueDate) {
		return "Which due date should the repeating task start on?"
	}
	message := err.Error()
	return strings.ToUpper(message[:1]) + message[1:] + "."
}

func (s *Server) prepareChatUpdateTask(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "change")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	_, changes, err := chatTaskPatch(action, task)
	var repeatErr chatRepeatError
	if errors.As(err, &repeatErr) {
		return chatAction{}, chatRepeatQuestion(repeatErr.error), false, nil
	}
	if err != nil {
		return chatAction{}, "", false, err
	}
	if len(changes) == 0 {
		return chatAction{}, "What should I change on @" + task.Ref + "?", false, nil
	}
	return action, "Change " + chatTaskLabel(task) + ": " + strings.Join(changes, ", ") + "?", true, nil
}

func (s *Server) confirmChatUpdateTask(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	patch, changes, err := chatTaskPatch(action, task)
	if err != nil {
		return "", err
	}
	if len(changes) == 0 {
		return "", fmt.Errorf("the update has no changes")
	}
	if _, err := s.Store.PatchTask(ctx, task.ID, patch); err != nil {
		return "", err
	}
	s.broadcast("tasks")
	return "Updated @" + task.Ref + ".", nil
}

func (s *Server) prepareChatCompleteTask(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "mark done")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	if task.CompletedAt != nil {
		return chatAction{}, "@" + task.Ref + " is already done.", false, nil
	}
	if _, found, err := s.doneColumn(ctx, task.BoardID); err != nil || !found {
		return chatAction{}, "The board for @" + task.Ref + " has no done column.", false, err
	}
	return action, "Mark " + chatTaskLabel(task) + " done?", true, nil
}

func (s *Server) confirmChatCompleteTask(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	column, found, err := s.doneColumn(ctx, task.BoardID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("the board has no done column")
	}
	if err := s.moveChatTask(ctx, task, column); err != nil {
		return "", err
	}
	return "Marked @" + task.Ref + " done.", nil
}

func (s *Server) prepareChatDeleteTask(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "delete")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	return action, "Delete " + chatTaskLabel(task) + "? You can restore it later.", true, nil
}

func (s *Server) confirmChatDeleteTask(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	if err := s.Store.ArchiveTask(ctx, task.ID); err != nil {
		return "", err
	}
	s.broadcast("tasks")
	return "Deleted @" + task.Ref + ".", nil
}

func tagByName(tags []store.Tag, name string) (store.Tag, bool) {
	for _, tag := range tags {
		if strings.EqualFold(tag.Name, name) {
			return tag, true
		}
	}
	return store.Tag{}, false
}

func (s *Server) prepareChatTagTask(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "tag")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	current, err := s.Store.ListTagsForTask(ctx, task.ID)
	if err != nil {
		return chatAction{}, "", false, err
	}
	var add, remove []string
	for _, name := range sanitiseLabels(action.Add) {
		if _, has := tagByName(current, name); !has {
			add = append(add, name)
		}
	}
	for _, name := range sanitiseLabels(action.Remove) {
		if tag, has := tagByName(current, name); has {
			remove = append(remove, tag.Name)
		}
	}
	if len(add) == 0 && len(remove) == 0 {
		return chatAction{}, "@" + task.Ref + " already has those tags.", false, nil
	}
	action.Add, action.Remove = add, remove
	var parts []string
	if len(add) > 0 {
		parts = append(parts, "add "+strings.Join(add, ", "))
	}
	if len(remove) > 0 {
		parts = append(parts, "remove "+strings.Join(remove, ", "))
	}
	return action, "Tag " + chatTaskLabel(task) + ": " + strings.Join(parts, "; ") + "?", true, nil
}

func (s *Server) confirmChatTagTask(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	tags, err := s.Store.ListTags(ctx)
	if err != nil {
		return "", err
	}
	// At most six tags per side; SQLite serialises the writes anyway.
	for _, name := range sanitiseLabels(action.Add) {
		tag, exists := tagByName(tags, name)
		if !exists {
			if tag, err = s.Store.CreateTag(ctx, name, "#888888"); err != nil {
				return "", err
			}
		}
		if err := s.Store.AddTaskTag(ctx, task.ID, tag.ID); err != nil {
			return "", err
		}
	}
	for _, name := range sanitiseLabels(action.Remove) {
		if tag, exists := tagByName(tags, name); exists {
			if err := s.Store.RemoveTaskTag(ctx, task.ID, tag.ID); err != nil {
				return "", err
			}
		}
	}
	s.broadcast("tasks")
	s.broadcast("tags")
	return "Updated tags on @" + task.Ref + ".", nil
}

func (s *Server) prepareChatComment(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "comment on")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	action.Body = strings.TrimSpace(action.Body)
	if action.Body == "" {
		return chatAction{}, "What should the comment on @" + task.Ref + " say?", false, nil
	}
	return action, "Comment on " + chatTaskLabel(task) + ": " + quoteSnippet(action.Body) + "?", true, nil
}

func (s *Server) confirmChatComment(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	body := strings.TrimSpace(action.Body)
	if body == "" {
		return "", fmt.Errorf("the comment is empty")
	}
	if _, err := s.Store.CreateTaskComment(ctx, task.ID, "", body, nil); err != nil {
		return "", err
	}
	s.broadcast("tasks")
	return "Commented on @" + task.Ref + ".", nil
}

const maxLoggedTime = 24 * time.Hour

func chatTimeSpan(action chatAction, now time.Time) (time.Time, time.Time, error) {
	end := now
	if action.EndedAt != "" {
		parsed, err := time.Parse(time.RFC3339, action.EndedAt)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid ended_at")
		}
		end = parsed
	}
	var start time.Time
	switch {
	case action.StartedAt != "":
		parsed, err := time.Parse(time.RFC3339, action.StartedAt)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid started_at")
		}
		start = parsed
	case action.Minutes > 0:
		start = end.Add(-time.Duration(action.Minutes) * time.Minute)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("log_time needs minutes or started_at")
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("the end time must be after the start time")
	}
	if end.Sub(start) > maxLoggedTime {
		return time.Time{}, time.Time{}, fmt.Errorf("a time entry cannot be longer than 24 hours")
	}
	if end.After(now.Add(time.Minute)) {
		return time.Time{}, time.Time{}, fmt.Errorf("a time entry cannot end in the future")
	}
	return start, end, nil
}

func formatLength(duration time.Duration) string {
	duration = duration.Round(time.Minute)
	hours, minutes := int(duration.Hours()), int(duration.Minutes())%60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatSpan(start, end time.Time) string {
	length := formatLength(end.Sub(start))
	endLayout := "15:04"
	if end.Format("2006-01-02") != start.Format("2006-01-02") {
		endLayout = "Mon 2006-01-02 15:04"
	}
	return length + ", " + start.Format("Mon 2006-01-02 15:04") + " to " + end.In(start.Location()).Format(endLayout)
}

func (s *Server) prepareChatLogTime(ctx context.Context, action chatAction, now time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "log time on")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	start, end, err := chatTimeSpan(action, now)
	if err != nil {
		return chatAction{}, "", false, err
	}
	action.StartedAt = start.Format(time.RFC3339)
	action.EndedAt = end.In(start.Location()).Format(time.RFC3339)
	action.Minutes = 0
	return action, "Log " + formatSpan(start, end) + " on " + chatTaskLabel(task) + "?", true, nil
}

func (s *Server) confirmChatLogTime(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	if action.StartedAt == "" || action.EndedAt == "" {
		return "", fmt.Errorf("the time entry needs a start and an end")
	}
	start, end, err := chatTimeSpan(action, time.Now())
	if err != nil {
		return "", err
	}
	if _, err := s.Store.LogTimeEntry(ctx, task.ID, start, end); err != nil {
		return "", err
	}
	s.broadcast("tasks")
	return "Logged " + formatSpan(start, end) + " on @" + task.Ref + ".", nil
}

func (s *Server) prepareChatStartTimer(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	task, question, err := s.targetChatTask(ctx, &action, "start the timer on")
	if err != nil || question != "" {
		return chatAction{}, question, false, err
	}
	running, err := s.Store.GetRunningEntry(ctx)
	if err != nil {
		return chatAction{}, "", false, err
	}
	if running == nil {
		return action, "Start the timer on " + chatTaskLabel(task) + "?", true, nil
	}
	if running.TaskID == task.ID {
		return chatAction{}, "The timer is already running on " + chatTaskLabel(task) + ".", false, nil
	}
	other, err := s.Store.GetTask(ctx, running.TaskID)
	if err != nil {
		return chatAction{}, "", false, err
	}
	return action, "Start the timer on " + chatTaskLabel(task) + "? This stops the timer on " + chatTaskLabel(other) + ".", true, nil
}

func (s *Server) confirmChatStartTimer(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	if _, err := s.Store.StartTimer(ctx, task.ID); err != nil {
		return "", err
	}
	s.broadcast("tasks")
	return "Started the timer on @" + task.Ref + ".", nil
}

func (s *Server) prepareChatStopTimer(ctx context.Context, action chatAction, _ time.Time) (chatAction, string, bool, error) {
	var task store.Task
	if action.Task != "" || action.TaskID != 0 {
		var question string
		var err error
		task, question, err = s.targetChatTask(ctx, &action, "stop the timer on")
		if err != nil || question != "" {
			return chatAction{}, question, false, err
		}
	}
	running, err := s.Store.GetRunningEntry(ctx)
	if err != nil {
		return chatAction{}, "", false, err
	}
	if running == nil {
		return chatAction{}, "No timer is running.", false, nil
	}
	if task.ID == 0 {
		if task, err = s.Store.GetTask(ctx, running.TaskID); err != nil {
			return chatAction{}, "", false, err
		}
		action.TaskID, action.TaskRef, action.TaskTitle = task.ID, task.Ref, task.Title
	} else if running.TaskID != task.ID {
		return chatAction{}, "No timer is running on " + chatTaskLabel(task) + ".", false, nil
	}
	return action, "Stop the timer on " + chatTaskLabel(task) + "?", true, nil
}

func (s *Server) confirmChatStopTimer(ctx context.Context, action chatAction) (string, error) {
	task, err := s.confirmedChatTask(ctx, action)
	if err != nil {
		return "", err
	}
	entry, err := s.Store.StopEntryForTask(ctx, task.ID)
	if errors.Is(err, store.ErrNotFound) {
		return "", fmt.Errorf("no timer is running on that task")
	}
	if err != nil {
		return "", err
	}
	s.broadcast("tasks")
	elapsed := time.Duration(0)
	if entry.DurationSeconds != nil {
		elapsed = time.Duration(*entry.DurationSeconds) * time.Second
	}
	return "Stopped the timer on @" + task.Ref + " after " + formatLength(elapsed) + ".", nil
}
