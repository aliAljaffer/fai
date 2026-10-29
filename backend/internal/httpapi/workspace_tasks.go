package httpapi

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const workspaceExportVersion = 1

type workspaceTasksExport struct {
	Version    int                   `json:"version"`
	Workspace  workspaceExportInfo   `json:"workspace"`
	ExportedAt time.Time             `json:"exported_at"`
	Tasks      []workspaceTaskExport `json:"tasks"`
}

type workspaceExportInfo struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type workspaceTaskExport struct {
	Ref            string               `json:"ref"`
	RefNum         int64                `json:"ref_num"`
	Board          string               `json:"board"`
	Column         string               `json:"column"`
	ColumnPosition int32                `json:"column_position"`
	ColumnIsDone   bool                 `json:"column_is_done"`
	Title          string               `json:"title"`
	DescriptionMd  string               `json:"description_md"`
	Priority       string               `json:"priority"`
	DueDate        *string              `json:"due_date"`
	Kind           string               `json:"kind"`
	ParentRef      *string              `json:"parent_ref"`
	Position       float64              `json:"position"`
	CompletedAt    *time.Time           `json:"completed_at"`
	Archived       bool                 `json:"archived"`
	Tags           []workspaceTagExport `json:"tags"`
}

type workspaceTagExport struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (s *Server) exportWorkspaceTasks(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseID(r, "workspaceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	workspace, err := s.Store.GetWorkspace(r.Context(), workspaceID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	boards, err := s.Store.ListBoards(r.Context(), &workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type entry struct {
		task       store.Task
		boardName  string
		columnName string
		columnPos  int32
		columnDone bool
	}
	entries := []entry{}
	refs := map[int64]string{}
	for _, board := range boards {
		columns, err := s.Store.ListColumnsByBoard(r.Context(), board.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		columnByID := map[int64]store.BoardColumn{}
		for _, column := range columns {
			columnByID[column.ID] = column
		}
		tasks, err := s.Store.ListTasks(r.Context(), store.ListTasksParams{BoardID: &board.ID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, task := range tasks {
			column := columnByID[task.ColumnID]
			entries = append(entries, entry{
				task:       task,
				boardName:  board.Name,
				columnName: column.Name,
				columnPos:  column.Position,
				columnDone: column.IsDone,
			})
			refs[task.ID] = task.Ref
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].task.RefNum < entries[j].task.RefNum
	})

	out := workspaceTasksExport{
		Version:    workspaceExportVersion,
		Workspace:  workspaceExportInfo{Name: workspace.Name, Tag: workspace.Tag},
		ExportedAt: time.Now().UTC(),
		Tasks:      make([]workspaceTaskExport, 0, len(entries)),
	}
	for _, item := range entries {
		var parentRef *string
		if item.task.ParentTaskID != nil {
			if ref, ok := refs[*item.task.ParentTaskID]; ok {
				parentRef = &ref
			}
		}
		tags := make([]workspaceTagExport, 0, len(item.task.Tags))
		for _, tag := range item.task.Tags {
			tags = append(tags, workspaceTagExport{Name: tag.Name, Color: tag.Color})
		}
		out.Tasks = append(out.Tasks, workspaceTaskExport{
			Ref:            item.task.Ref,
			RefNum:         item.task.RefNum,
			Board:          item.boardName,
			Column:         item.columnName,
			ColumnPosition: item.columnPos,
			ColumnIsDone:   item.columnDone,
			Title:          item.task.Title,
			DescriptionMd:  item.task.DescriptionMd,
			Priority:       item.task.Priority,
			DueDate:        item.task.DueDate,
			Kind:           item.task.Kind,
			ParentRef:      parentRef,
			Position:       item.task.Position,
			CompletedAt:    item.task.CompletedAt,
			Archived:       item.task.Archived,
			Tags:           tags,
		})
	}

	filename := fmt.Sprintf("fai-%s-tasks-%s.json", strings.ToLower(workspace.Tag), time.Now().UTC().Format("20060102"))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) importWorkspaceTasks(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseID(r, "workspaceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	if _, err := s.Store.GetWorkspace(r.Context(), workspaceID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "workspace not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var body workspaceTasksExport
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Version != 0 && body.Version != workspaceExportVersion {
		writeError(w, http.StatusBadRequest, "unsupported export version")
		return
	}
	for i, item := range body.Tasks {
		if strings.TrimSpace(item.Title) == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("task %d has no title", i+1))
			return
		}
	}

	boards, err := s.Store.ListBoards(r.Context(), &workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	boardByName := map[string]store.Board{}
	for _, board := range boards {
		boardByName[strings.ToLower(strings.TrimSpace(board.Name))] = board
	}

	existingRefs := map[int64]bool{}
	for _, board := range boards {
		tasks, err := s.Store.ListTasks(r.Context(), store.ListTasksParams{BoardID: &board.ID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, task := range tasks {
			existingRefs[task.RefNum] = true
		}
	}
	importedRefs := map[int64]bool{}
	for i, item := range body.Tasks {
		if item.RefNum <= 0 {
			continue
		}
		if existingRefs[item.RefNum] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("task %d uses number %d, which already exists", i+1, item.RefNum))
			return
		}
		if importedRefs[item.RefNum] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("task %d repeats number %d", i+1, item.RefNum))
			return
		}
		importedRefs[item.RefNum] = true
	}

	type columnSet struct {
		byName   map[string]store.BoardColumn
		usedPos  map[int32]bool
		nextPos  int32
		first    store.BoardColumn
		hasFirst bool
	}
	columnsByBoard := map[int64]*columnSet{}
	loadColumns := func(boardID int64) (*columnSet, error) {
		if set, ok := columnsByBoard[boardID]; ok {
			return set, nil
		}
		columns, err := s.Store.ListColumnsByBoard(r.Context(), boardID)
		if err != nil {
			return nil, err
		}
		set := &columnSet{byName: map[string]store.BoardColumn{}, usedPos: map[int32]bool{}}
		for _, column := range columns {
			set.byName[strings.ToLower(strings.TrimSpace(column.Name))] = column
			set.usedPos[column.Position] = true
			if column.Position >= set.nextPos {
				set.nextPos = column.Position + 1
			}
			if !set.hasFirst || column.Position < set.first.Position {
				set.first = column
				set.hasFirst = true
			}
		}
		columnsByBoard[boardID] = set
		return set, nil
	}

	tags, err := s.Store.ListTags(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tagByName := map[string]store.Tag{}
	for _, tag := range tags {
		tagByName[strings.ToLower(tag.Name)] = tag
	}

	newTaskIDs := map[string]int64{}
	imported := 0
	for i, item := range body.Tasks {
		boardName := strings.TrimSpace(item.Board)
		var board store.Board
		if boardName == "" && len(boards) > 0 {
			board = boards[0]
		} else {
			if boardName == "" {
				boardName = "Imported"
			}
			var found bool
			board, found = boardByName[strings.ToLower(boardName)]
			if !found {
				board, err = s.Store.CreateBoard(r.Context(), boardName, workspaceID)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				boards = append(boards, board)
				boardByName[strings.ToLower(boardName)] = board
			}
		}

		set, err := loadColumns(board.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		columnName := strings.TrimSpace(item.Column)
		column, found := set.byName[strings.ToLower(columnName)]
		if columnName == "" {
			if set.hasFirst {
				column, found = set.first, true
			} else {
				columnName = "Imported"
			}
		}
		if !found {
			position := item.ColumnPosition
			if position < 0 || set.usedPos[position] {
				position = set.nextPos
			}
			column, err = s.Store.CreateColumn(r.Context(), board.ID, columnName, position, item.ColumnIsDone)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			set.byName[strings.ToLower(columnName)] = column
			set.usedPos[position] = true
			set.nextPos = position + 1
			if !set.hasFirst || column.Position < set.first.Position {
				set.first = column
				set.hasFirst = true
			}
		}

		priority := item.Priority
		if priority == "" {
			priority = "medium"
		}
		var refNum *int64
		if item.RefNum > 0 {
			refNum = &item.RefNum
		}
		task, err := s.Store.CreateTask(r.Context(), store.CreateTaskParams{
			BoardID:       board.ID,
			ColumnID:      column.ID,
			Title:         item.Title,
			DescriptionMd: item.DescriptionMd,
			Priority:      priority,
			DueDate:       item.DueDate,
			Position:      item.Position,
			Kind:          item.Kind,
			RefNum:        refNum,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("task %d: %v", i+1, err))
			return
		}
		if item.Ref != "" {
			newTaskIDs[item.Ref] = task.ID
		}

		for _, exportedTag := range item.Tags {
			name := strings.TrimSpace(exportedTag.Name)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			tag, found := tagByName[key]
			if !found {
				color := exportedTag.Color
				if color == "" {
					color = "#888888"
				}
				tag, err = s.Store.CreateTag(r.Context(), name, color)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				tagByName[key] = tag
			}
			if err := s.Store.AddTaskTag(r.Context(), task.ID, tag.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		if item.CompletedAt != nil {
			if err := s.Store.SetTaskCompletedAt(r.Context(), task.ID, *item.CompletedAt); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if item.Archived {
			if err := s.Store.ArchiveTask(r.Context(), task.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		imported++
	}

	for _, item := range body.Tasks {
		if item.ParentRef == nil || *item.ParentRef == "" {
			continue
		}
		parentID, ok := newTaskIDs[*item.ParentRef]
		if !ok {
			continue
		}
		childID, ok := newTaskIDs[item.Ref]
		if !ok {
			continue
		}
		if err := s.Store.SetTaskParent(r.Context(), childID, &parentID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
}

type resetWorkspaceBody struct {
	Confirm string `json:"confirm"`
}

func (s *Server) resetWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseID(r, "workspaceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	var body resetWorkspaceBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(body.Confirm) != "I understand" {
		writeError(w, http.StatusBadRequest, `type "I understand" to confirm`)
		return
	}

	deleted, storageKeys, err := s.Store.ResetWorkspaceTasks(r.Context(), workspaceID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, key := range storageKeys {
		path, err := s.attachmentPath(key)
		if err != nil {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("attachments: removing %s: %v", key, err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
}
