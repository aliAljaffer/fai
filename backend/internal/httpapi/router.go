package httpapi

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/jira"
	"github.com/alialjaffer/fai/backend/internal/store"
)

type Server struct {
	Store                store.Store
	Jira                 *jira.Client
	JiraDefaultProject   string
	JiraStatusColumnMap  map[string]string
	JiraEpicIssueType    string
	JiraTaskIssueType    string
	JiraSubtaskIssueType string
	JiraEpicLinkField    string
	AI                   ai.Provider
	// Environment-provided AI configuration, used as the default until the
	// owner saves model settings in the database.
	AIProvider     string
	AIBaseURL      string
	AIModel        string
	AIAPIKey       string
	ICSToken       string
	PublicURL      string
	AttachmentsDir string
	// StaticDir, when set, serves the built SPA from the API server. The
	// Electron build uses this so the window can load a normal http origin.
	StaticDir string
	// AuthMiddleware validates bearer tokens on every /api/* route when set.
	// Left nil under DISABLE_AUTH for local dev.
	AuthMiddleware func(http.Handler) http.Handler

	// Last poll result per calendar source, so a failed background sync is
	// visible in the UI even while cached events are served.
	syncMu      sync.Mutex
	calendarErr map[int64]string
}

func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(securityHeaders)
	// The calendar feed carries its credential in the query string; keep it out
	// of the access log without touching the handler's view of the request.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/calendar.ics" {
				r.RequestURI = r.URL.Path
			}
			next.ServeHTTP(w, r)
		})
	})
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		// Calendar clients cannot complete OIDC; this one route is gated by its
		// own URL token instead (see calendar.go).
		r.Get("/calendar.ics", s.calendarFeed)

		r.Group(func(r chi.Router) {
			if s.AuthMiddleware != nil {
				r.Use(s.AuthMiddleware)
			}
			r.Use(rateLimitByUser(600, 120))

			r.Get("/calendar/feed", s.calendarFeedInfo)
			r.Get("/calendar/sources", s.listCalendarSources)
			r.Post("/calendar/sources", s.createCalendarSource)
			r.Patch("/calendar/sources/{sourceID}", s.updateCalendarSource)
			r.Delete("/calendar/sources/{sourceID}", s.deleteCalendarSource)
			r.Get("/calendar/imported", s.listImportedCalendars)
			r.Get("/workspaces", s.listWorkspaces)
			r.Post("/workspaces", s.createWorkspace)
			r.Patch("/workspaces/{workspaceID}", s.updateWorkspace)
			r.Get("/workspaces/{workspaceID}/tasks/export", s.exportWorkspaceTasks)
			r.Post("/workspaces/{workspaceID}/tasks/import", s.importWorkspaceTasks)
			r.Post("/workspaces/{workspaceID}/reset", s.resetWorkspace)
			r.Get("/backup", s.exportBackup)
			r.Post("/backup/restore", s.restoreBackup)
			r.Get("/refs/resolve", s.resolveRefs)

			r.Get("/boards", s.listBoards)
			r.Post("/boards", s.createBoard)
			r.Patch("/boards/{boardID}", s.updateBoard)
			r.Get("/boards/{boardID}/columns", s.listColumns)
			r.Post("/boards/{boardID}/columns", s.createColumn)
			r.Patch("/boards/{boardID}/columns", s.reorderColumns)
			r.Patch("/boards/{boardID}/columns/{columnID}", s.updateColumn)

			r.Get("/tasks", s.listTasks)
			r.Post("/tasks", s.createTask)
			r.Get("/tasks/{taskID}", s.getTask)
			r.Get("/tasks/{taskID}/subtasks", s.listSubtasks)
			r.Get("/tasks/{taskID}/comments", s.listTaskComments)
			r.Post("/tasks/{taskID}/comments", s.createTaskComment)
			r.Delete("/tasks/{taskID}/comments/{commentID}", s.deleteTaskComment)
			r.Get("/tasks/{taskID}/references", s.listTaskReferences)
			r.Patch("/tasks/{taskID}", s.updateTask)
			r.Patch("/tasks/{taskID}/type", s.setTaskType)
			r.Delete("/tasks/{taskID}", s.deleteTask)
			r.Post("/tasks/{taskID}/move", s.moveTask)
			r.Put("/tasks/{taskID}/tags/{tagID}", s.addTaskTag)
			r.Delete("/tasks/{taskID}/tags/{tagID}", s.removeTaskTag)

			r.Get("/tasks/{taskID}/attachments", s.listTaskAttachments)
			r.Post("/tasks/{taskID}/attachments", s.uploadTaskAttachment)
			r.Get("/notes/{noteID}/attachments", s.listNoteAttachments)
			r.Post("/notes/{noteID}/attachments", s.uploadNoteAttachment)
			r.Get("/attachments/{attachmentID}", s.downloadAttachment)
			r.Delete("/attachments/{attachmentID}", s.deleteAttachment)

			r.Get("/tags", s.listTags)
			r.Post("/tags", s.createTag)

			r.Get("/search", s.search)
			r.Get("/stats/completions", s.statsCompletions)
			r.Get("/stats/time", s.statsTime)

			r.Post("/tasks/{taskID}/time/start", s.startTime)
			r.Post("/tasks/{taskID}/time/stop", s.stopTime)
			r.Get("/tasks/{taskID}/time", s.listTimeForTask)
			r.Get("/time/running", s.runningTime)

			r.Get("/notes", s.listNotes)
			r.Post("/notes", s.createNote)
			r.Get("/notes/by-title/{title}", s.getNoteByTitle)
			r.Get("/notes/{noteID}", s.getNote)
			r.Patch("/notes/{noteID}", s.updateNote)
			r.Delete("/notes/{noteID}", s.deleteNote)
			r.Post("/notes/{noteID}/tasks", s.createTaskFromNote)

			r.Get("/reminders", s.listReminders)
			r.Post("/reminders", s.createReminder)
			r.Patch("/reminders/{reminderID}", s.updateReminder)
			r.Delete("/reminders/{reminderID}", s.deleteReminder)

			r.Get("/jira/links", s.jiraLinks)
			r.Post("/jira/import", s.jiraImport)
			r.Post("/jira/export", s.jiraExport)
			r.Get("/tasks/{taskID}/jira/link", s.jiraGetLink)
			r.Post("/tasks/{taskID}/jira/link", s.jiraLink)
			r.Delete("/tasks/{taskID}/jira/link", s.jiraUnlink)
			r.Get("/tasks/{taskID}/jira/suggestions", s.jiraSuggestions)

			r.Get("/ai/status", s.aiStatus)
			r.Get("/ai/settings", s.getAISettings)
			r.Put("/ai/settings", s.updateAISettings)
			r.Get("/appearance", s.getAppearance)
			r.Put("/appearance", s.updateAppearance)
			r.Group(func(r chi.Router) {
				r.Use(rateLimitByUser(10, 5))
				r.Post("/tasks/{taskID}/ai/breakdown", s.aiBreakdown)
				r.Post("/tasks/{taskID}/ai/labels", s.aiSuggestLabels)
				r.Post("/ai/chat", s.aiChat)
				r.Post("/meeting-outcomes/{outcomeID}/ai/extract-tasks", s.extractTasksFromOutcome)
				r.Post("/notes/{noteID}/ai/plan", s.planNoteTask)
				r.Post("/reminders/ai", s.aiReminder)
				r.Post("/tasks/ai", s.aiTask)
			})

			r.Get("/meeting-outcomes", s.listMeetingOutcomes)
			r.Post("/meeting-outcomes", s.createMeetingOutcome)
			r.Get("/meeting-outcomes/{outcomeID}", s.getMeetingOutcome)
			r.Patch("/meeting-outcomes/{outcomeID}", s.updateMeetingOutcome)
			r.Delete("/meeting-outcomes/{outcomeID}", s.deleteMeetingOutcome)
			r.Post("/meeting-outcomes/{outcomeID}/tasks", s.createTaskFromOutcome)
		})
	})

	if s.StaticDir != "" {
		r.Handle("/*", spaHandler(s.StaticDir))
	}

	return r
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
