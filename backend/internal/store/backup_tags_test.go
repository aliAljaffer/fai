package store

import (
	"reflect"
	"testing"
)

func TestScopeTagsToWorkspaces(t *testing.T) {
	data := BackupData{
		Workspaces: []BackupWorkspace{{ID: 2}, {ID: 1}},
		Tasks:      []BackupTask{{ID: 10, WorkspaceID: 1}, {ID: 20, WorkspaceID: 2}},
		Notes:      []BackupNote{{ID: 30, WorkspaceID: 2}},
		Tags: []BackupTag{
			{ID: 1, Name: "shared", Color: "#111111"},
			{ID: 2, Name: "unused", Color: "#222222"},
			{ID: 5, Name: "note-only", Color: "#555555"},
		},
		TaskTags: []BackupTaskTag{{TaskID: 10, TagID: 1}, {TaskID: 20, TagID: 1}},
		NoteTags: []BackupNoteTag{{NoteID: 30, TagID: 5}, {NoteID: 30, TagID: 1}},
	}

	data.ScopeTagsToWorkspaces()

	wantTags := []BackupTag{
		{ID: 1, WorkspaceID: 1, Name: "shared", Color: "#111111"},
		{ID: 2, WorkspaceID: 1, Name: "unused", Color: "#222222"},
		{ID: 5, WorkspaceID: 2, Name: "note-only", Color: "#555555"},
		{ID: 6, WorkspaceID: 2, Name: "shared", Color: "#111111"},
	}
	if !reflect.DeepEqual(data.Tags, wantTags) {
		t.Errorf("tags = %#v", data.Tags)
	}
	wantTaskTags := []BackupTaskTag{{TaskID: 10, TagID: 1}, {TaskID: 20, TagID: 6}}
	if !reflect.DeepEqual(data.TaskTags, wantTaskTags) {
		t.Errorf("task tags = %#v", data.TaskTags)
	}
	wantNoteTags := []BackupNoteTag{{NoteID: 30, TagID: 5}, {NoteID: 30, TagID: 6}}
	if !reflect.DeepEqual(data.NoteTags, wantNoteTags) {
		t.Errorf("note tags = %#v", data.NoteTags)
	}
}
