package store

import "slices"

// ScopeTagsToWorkspaces converts tags shared by every workspace into tags
// owned by one workspace. A tag keeps its id in the lowest workspace that uses
// it, or in the first workspace when nothing uses it. Every other workspace
// that uses it gets a copy with a new id.
func (d *BackupData) ScopeTagsToWorkspaces() {
	taskWorkspace := make(map[int64]int64, len(d.Tasks))
	for _, task := range d.Tasks {
		taskWorkspace[task.ID] = task.WorkspaceID
	}
	noteWorkspace := make(map[int64]int64, len(d.Notes))
	for _, note := range d.Notes {
		noteWorkspace[note.ID] = note.WorkspaceID
	}

	usedIn := map[int64][]int64{}
	use := func(tagID, workspaceID int64) {
		if workspaceID != 0 && !slices.Contains(usedIn[tagID], workspaceID) {
			usedIn[tagID] = append(usedIn[tagID], workspaceID)
		}
	}
	for _, link := range d.TaskTags {
		use(link.TagID, taskWorkspace[link.TaskID])
	}
	for _, link := range d.NoteTags {
		use(link.TagID, noteWorkspace[link.NoteID])
	}

	var firstWorkspace, nextID int64
	for i, workspace := range d.Workspaces {
		if i == 0 || workspace.ID < firstWorkspace {
			firstWorkspace = workspace.ID
		}
	}
	for _, tag := range d.Tags {
		nextID = max(nextID, tag.ID)
	}

	type scope struct{ tagID, workspaceID int64 }
	scoped := map[scope]int64{}
	shared := len(d.Tags)
	for i := range shared {
		tag := &d.Tags[i]
		workspaces := usedIn[tag.ID]
		slices.Sort(workspaces)
		if len(workspaces) == 0 {
			tag.WorkspaceID = firstWorkspace
			continue
		}
		tag.WorkspaceID = workspaces[0]
		for _, workspaceID := range workspaces[1:] {
			nextID++
			scoped[scope{tag.ID, workspaceID}] = nextID
			d.Tags = append(d.Tags, BackupTag{ID: nextID, WorkspaceID: workspaceID, Name: tag.Name, Color: tag.Color})
			tag = &d.Tags[i]
		}
	}

	for i, link := range d.TaskTags {
		if id, ok := scoped[scope{link.TagID, taskWorkspace[link.TaskID]}]; ok {
			d.TaskTags[i].TagID = id
		}
	}
	for i, link := range d.NoteTags {
		if id, ok := scoped[scope{link.TagID, noteWorkspace[link.NoteID]}]; ok {
			d.NoteTags[i].TagID = id
		}
	}
}
