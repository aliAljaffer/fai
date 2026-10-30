package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/alialjaffer/fai/backend/internal/store"
)

type fakeSource struct {
	tasks       []store.Task
	notes       []store.Note
	reminders   []store.Reminder
	completions []store.DayCount
}

func (f fakeSource) ListTasks(context.Context, store.ListTasksParams) ([]store.Task, error) {
	return f.tasks, nil
}

func (f fakeSource) ListNotes(context.Context, *int64) ([]store.Note, error) {
	return f.notes, nil
}

func (f fakeSource) ListReminders(context.Context, *int64) ([]store.Reminder, error) {
	return f.reminders, nil
}

func (f fakeSource) CompletionsByDay(context.Context, string, string) ([]store.DayCount, error) {
	return f.completions, nil
}

func TestBusinessMetrics(t *testing.T) {
	now := time.Now()
	due := now.AddDate(0, 0, -1).Format("2006-01-02")
	done := now.Add(-time.Hour)
	m := New(fakeSource{
		tasks: []store.Task{
			{Priority: "high", DueDate: &due},
			{Priority: "low"},
			{Priority: "low", CompletedAt: &done},
		},
		notes:     []store.Note{{}, {}},
		reminders: []store.Reminder{{}, {CompletedAt: &done}},
		completions: []store.DayCount{
			{Count: 2}, {Count: 1},
		},
	})

	for name, want := range map[string]float64{
		`fai_tasks{state="open"}`:                2,
		`fai_tasks{state="completed"}`:           1,
		`fai_tasks_overdue`:                      1,
		`fai_tasks_by_priority{priority="high"}`: 1,
		`fai_tasks_by_priority{priority="low"}`:  1,
		`fai_tasks_completed_7d`:                 3,
		`fai_notes_total`:                        2,
		`fai_reminders_pending`:                  1,
		`fai_business_metrics_up`:                1,
	} {
		if got, want := gatherValue(t, m, name), want; got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
}

// gatherValue reads one metric from the registry by its exposition line prefix.
func gatherValue(t *testing.T, m *Metrics, name string) float64 {
	t.Helper()
	families, err := m.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		for _, metric := range f.GetMetric() {
			var labels []string
			for _, l := range metric.GetLabel() {
				labels = append(labels, l.GetName()+`="`+l.GetValue()+`"`)
			}
			line := f.GetName()
			if len(labels) > 0 {
				line += "{" + strings.Join(labels, ",") + "}"
			}
			if line == name {
				return metric.GetGauge().GetValue()
			}
		}
	}
	t.Fatalf("metric %s not found", name)
	return 0
}

func TestHTTPMiddlewareUsesRoutePattern(t *testing.T) {
	m := New(fakeSource{})
	r := chi.NewRouter()
	r.Use(m.Middleware)
	r.Get("/api/tasks/{taskID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/tasks/42", nil))

	if got := testutil.ToFloat64(m.requests.WithLabelValues(http.MethodGet, "/api/tasks/{taskID}", "418")); got != 1 {
		t.Fatalf("request counter = %v, want 1", got)
	}
}
