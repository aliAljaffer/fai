package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

type Server struct {
	Q *sqlc.Queries
}

func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/boards", s.listBoards)
		r.Post("/boards", s.createBoard)
		r.Get("/boards/{boardID}/columns", s.listColumns)
		r.Post("/boards/{boardID}/columns", s.createColumn)

		r.Get("/tasks", s.listTasks)
		r.Post("/tasks", s.createTask)
		r.Get("/tasks/{taskID}", s.getTask)
		r.Patch("/tasks/{taskID}", s.updateTask)
		r.Delete("/tasks/{taskID}", s.deleteTask)
		r.Post("/tasks/{taskID}/move", s.moveTask)
		r.Put("/tasks/{taskID}/tags/{tagID}", s.addTaskTag)
		r.Delete("/tasks/{taskID}/tags/{tagID}", s.removeTaskTag)

		r.Get("/tags", s.listTags)
		r.Post("/tags", s.createTag)

		r.Get("/search", s.search)

		r.Get("/notes", s.listNotes)
		r.Post("/notes", s.createNote)
		r.Get("/notes/by-title/{title}", s.getNoteByTitle)
		r.Get("/notes/{noteID}", s.getNote)
		r.Patch("/notes/{noteID}", s.updateNote)
		r.Delete("/notes/{noteID}", s.deleteNote)
	})

	return r
}
