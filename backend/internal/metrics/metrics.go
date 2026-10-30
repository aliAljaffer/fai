// Package metrics exposes Prometheus metrics for the fai backend: HTTP request
// metrics for every route, plus business gauges read from the store on scrape.
package metrics

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/alialjaffer/fai/backend/internal/store"
)

// Source is the subset of the store the business gauges need.
type Source interface {
	ListTasks(ctx context.Context, p store.ListTasksParams) ([]store.Task, error)
	ListNotes(ctx context.Context, workspaceID *int64) ([]store.Note, error)
	ListReminders(ctx context.Context, workspaceID *int64) ([]store.Reminder, error)
	CompletionsByDay(ctx context.Context, from, to string) ([]store.DayCount, error)
}

type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

func New(src Source) *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "fai_http_requests_total",
			Help: "HTTP requests handled, by method, route pattern and status code.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "fai_http_request_duration_seconds",
			Help:    "HTTP request latency, by method and route pattern.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "fai_http_requests_in_flight",
			Help: "HTTP requests currently being served.",
		}),
	}
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewBuildInfoCollector(),
		m.requests, m.duration, m.inFlight,
		newBusiness(src),
	)
	return m
}

// Handler serves the metrics in the Prometheus text exposition format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware records every routed request. The chi route pattern ("/api/tasks/{taskID}")
// keeps label cardinality bounded.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		// chi fills the route pattern during routing, so read it after the call.
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
		m.duration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

type business struct {
	src          Source
	tasks        *prometheus.Desc
	overdue      *prometheus.Desc
	byPriority   *prometheus.Desc
	completed7d  *prometheus.Desc
	notes        *prometheus.Desc
	reminders    *prometheus.Desc
	scrapeFailed *prometheus.Desc
}

func newBusiness(src Source) *business {
	return &business{
		src: src,
		tasks: prometheus.NewDesc("fai_tasks",
			"Tasks by state (open, completed), excluding archived.", []string{"state"}, nil),
		overdue: prometheus.NewDesc("fai_tasks_overdue",
			"Open tasks whose due date has passed.", nil, nil),
		byPriority: prometheus.NewDesc("fai_tasks_by_priority",
			"Open tasks by priority.", []string{"priority"}, nil),
		completed7d: prometheus.NewDesc("fai_tasks_completed_7d",
			"Tasks completed in the last 7 days.", nil, nil),
		notes: prometheus.NewDesc("fai_notes_total",
			"Stored notes.", nil, nil),
		reminders: prometheus.NewDesc("fai_reminders_pending",
			"Reminders not yet completed.", nil, nil),
		scrapeFailed: prometheus.NewDesc("fai_business_metrics_up",
			"Whether the business gauges were collected from the database on this scrape.", nil, nil),
	}
}

func (b *business) Describe(ch chan<- *prometheus.Desc) {
	ch <- b.tasks
	ch <- b.overdue
	ch <- b.byPriority
	ch <- b.completed7d
	ch <- b.notes
	ch <- b.reminders
	ch <- b.scrapeFailed
}

func (b *business) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasks, err := b.src.ListTasks(ctx, store.ListTasksParams{})
	if err != nil {
		ch <- prometheus.MustNewConstMetric(b.scrapeFailed, prometheus.GaugeValue, 0)
		log.Printf("metrics: listing tasks: %v", err)
		return
	}
	notes, err := b.src.ListNotes(ctx, nil)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(b.scrapeFailed, prometheus.GaugeValue, 0)
		log.Printf("metrics: listing notes: %v", err)
		return
	}
	reminders, err := b.src.ListReminders(ctx, nil)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(b.scrapeFailed, prometheus.GaugeValue, 0)
		log.Printf("metrics: listing reminders: %v", err)
		return
	}
	now := time.Now()
	completions, err := b.src.CompletionsByDay(ctx, now.AddDate(0, 0, -7).Format("2006-01-02"), now.Format("2006-01-02"))
	if err != nil {
		ch <- prometheus.MustNewConstMetric(b.scrapeFailed, prometheus.GaugeValue, 0)
		log.Printf("metrics: completions: %v", err)
		return
	}

	today := now.Format("2006-01-02")
	open, completed, overdue, completed7d := 0, 0, 0, 0
	priority := map[string]int{}
	for _, t := range tasks {
		if t.CompletedAt != nil {
			completed++
			continue
		}
		open++
		priority[t.Priority]++
		if t.DueDate != nil && *t.DueDate < today {
			overdue++
		}
	}
	for _, c := range completions {
		completed7d += int(c.Count)
	}
	pending := 0
	for _, r := range reminders {
		if r.CompletedAt == nil {
			pending++
		}
	}

	ch <- prometheus.MustNewConstMetric(b.tasks, prometheus.GaugeValue, float64(open), "open")
	ch <- prometheus.MustNewConstMetric(b.tasks, prometheus.GaugeValue, float64(completed), "completed")
	ch <- prometheus.MustNewConstMetric(b.overdue, prometheus.GaugeValue, float64(overdue))
	for p, n := range priority {
		ch <- prometheus.MustNewConstMetric(b.byPriority, prometheus.GaugeValue, float64(n), p)
	}
	ch <- prometheus.MustNewConstMetric(b.completed7d, prometheus.GaugeValue, float64(completed7d))
	ch <- prometheus.MustNewConstMetric(b.notes, prometheus.GaugeValue, float64(len(notes)))
	ch <- prometheus.MustNewConstMetric(b.reminders, prometheus.GaugeValue, float64(pending))
	ch <- prometheus.MustNewConstMetric(b.scrapeFailed, prometheus.GaugeValue, 1)
}
