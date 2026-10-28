package pg

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) attachTags(ctx context.Context, tasks []store.Task) error {
	for i := range tasks {
		tasks[i].Tags = []store.Tag{}
	}
	if len(tasks) == 0 {
		return nil
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

func (s *Store) ListTagsForTask(ctx context.Context, taskID int64) ([]store.Tag, error) {
	tasks := []store.Task{{ID: taskID}}
	if err := s.attachTags(ctx, tasks); err != nil {
		return nil, err
	}
	return tasks[0].Tags, nil
}
