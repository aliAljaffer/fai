package httpapi

import (
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": []any{}, "notes": []any{}})
		return
	}

	workspaceID := optionalInt64Param(r, "workspace_id")
	var tasks []store.Task
	var notes []store.Note
	g, ctx := errgroup.WithContext(r.Context())
	g.Go(func() error {
		var err error
		tasks, err = s.Store.SearchTasks(ctx, q, workspaceID)
		return err
	})
	g.Go(func() error {
		var err error
		notes, err = s.Store.SearchNotes(ctx, q, workspaceID)
		return err
	})
	if err := g.Wait(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks, "notes": notes})
}
