package pg

import (
	"context"
	"strconv"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) attachTags(ctx context.Context, tasks []store.Task) error {
	for i := range tasks {
		tasks[i].Tags = []store.Tag{}
	}
	if len(tasks) == 0 {
		return nil
	}

	tagsByWorkspace, err := s.workspaceTagsByID(ctx)
	if err != nil {
		return err
	}
	for i := range tasks {
		tasks[i].Ref = tagsByWorkspace[tasks[i].WorkspaceID] + "-" + strconv.FormatInt(tasks[i].RefNum, 10)
	}

	ids := make([]int64, len(tasks))
	index := make(map[int64]int, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
		index[t.ID] = i
	}

	rows, err := s.pool.Query(ctx, `
		SELECT tt.task_id, t.id, t.name, t.color
		FROM task_tags tt
		JOIN tags t ON t.id = tt.tag_id
		WHERE tt.task_id = ANY($1::bigint[])
		ORDER BY t.name`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var taskID int64
		var tag store.Tag
		if err := rows.Scan(&taskID, &tag.ID, &tag.Name, &tag.Color); err != nil {
			return err
		}
		i, ok := index[taskID]
		if !ok {
			continue
		}
		tasks[i].Tags = append(tasks[i].Tags, tag)
	}
	return rows.Err()
}

func (s *Store) workspaceTagsByID(ctx context.Context) (map[int64]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, tag FROM workspaces`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return nil, err
		}
		out[id] = tag
	}
	return out, rows.Err()
}

func (s *Store) ListTagsForTask(ctx context.Context, taskID int64) ([]store.Tag, error) {
	tasks := []store.Task{{ID: taskID}}
	if err := s.attachTags(ctx, tasks); err != nil {
		return nil, err
	}
	return tasks[0].Tags, nil
}
