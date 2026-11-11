package httpapi

type chatCapability struct {
	Summary string
	Actions []string
}

var chatCapabilities = []chatCapability{
	{"Create tasks (including repeating ones), arcs, and steps", []string{"create_task", "create_arc", "add_steps"}},
	{"Find, edit, move, complete, and delete tasks", []string{"find_task", "update_task", "move_task", "complete_task", "delete_task"}},
	{"Tag tasks and comment on them", []string{"tag_task", "comment_task"}},
	{"Log time on a task and start or stop its timer", []string{"log_time", "start_timer", "stop_timer"}},
	{"Create, edit, complete, and delete reminders (including repeating ones)", []string{"create_reminder", "update_reminder", "complete_reminder", "delete_reminder"}},
	{"Create notes and add to them", []string{"create_note", "append_note"}},
}

func ChatCapabilities() []string {
	summaries := make([]string, len(chatCapabilities))
	for i, capability := range chatCapabilities {
		summaries[i] = capability.Summary
	}
	return summaries
}
