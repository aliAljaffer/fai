package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func BenchmarkCalendarFeed(b *testing.B) {
	st, err := lite.New(filepath.Join(b.TempDir(), "fai.db"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(st.Close)
	ctx := context.Background()
	for i := 0; i < 2000; i++ {
		params := store.CreateTaskParams{
			BoardID: 1, ColumnID: 1, Title: fmt.Sprintf("Task %d", i), Priority: "medium", Position: float64(i + 1),
		}
		if i%2 == 0 {
			due := "2026-12-01"
			params.DueDate = &due
		}
		if _, err := st.CreateTask(ctx, params); err != nil {
			b.Fatal(err)
		}
	}
	for i := 0; i < 300; i++ {
		if _, err := st.CreateReminder(ctx, fmt.Sprintf("Reminder %d", i), time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC).Add(time.Duration(i)*time.Hour), nil, 0, nil); err != nil {
			b.Fatal(err)
		}
	}
	server := &Server{Store: st, ICSToken: "secret", PublicURL: "https://fai.example"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.calendarFeed(rec, httptest.NewRequest(http.MethodGet, "/api/calendar.ics?token=secret", nil))
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d", rec.Code)
		}
	}
}
