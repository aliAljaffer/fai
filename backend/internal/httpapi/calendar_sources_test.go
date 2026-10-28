package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func TestNormalizeCalendarURL(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{
			name: "webcal becomes https",
			in:   "webcal://outlook.office365.com/owa/calendar/abc/calendar.ics",
			want: "https://outlook.office365.com/owa/calendar/abc/calendar.ics",
		},
		{
			name: "https stays",
			in:   "https://example.com/calendar.ics",
			want: "https://example.com/calendar.ics",
		},
		{
			name: "http stays",
			in:   "http://example.com/calendar.ics",
			want: "http://example.com/calendar.ics",
		},
		{
			name:    "other scheme rejected",
			in:      "ftp://example.com/calendar.ics",
			wantErr: true,
		},
		{
			name:    "missing host rejected",
			in:      "https:///calendar.ics",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeCalendarURL(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestImportedCalendarsSyncAndCache(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	const body = "BEGIN:VCALENDAR\r\n" +
		"VERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:outlook-1\r\n" +
		"SUMMARY:Outlook event\r\n" +
		"DTSTART;VALUE=DATE:20261015\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	var hits atomic.Int32
	feed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/calendar")
		w.Write([]byte(body))
	}))

	ctx := context.Background()
	source, err := st.CreateCalendarSource(ctx, "Outlook", feed.URL, "#2563eb")
	if err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st}
	server := httptest.NewServer(NewRouter(srv))
	defer server.Close()

	fetch := func() []importedCalendar {
		res, err := http.Get(server.URL + "/api/calendar/imported?from=2026-10-01&to=2026-11-01")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var calendars []importedCalendar
		if err := json.NewDecoder(res.Body).Decode(&calendars); err != nil {
			t.Fatal(err)
		}
		return calendars
	}

	first := fetch()
	if len(first) != 1 || first[0].Error != "" || len(first[0].Events) != 1 {
		t.Fatalf("unexpected first result: %+v", first)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("first view fetched %d times, want 1", got)
	}

	fetch()
	if got := hits.Load(); got != 1 {
		t.Fatalf("cached view fetched again (%d), want 1", got)
	}

	stored, _, ok, err := st.GetCalendarFeed(ctx, source.ID)
	if err != nil || !ok || len(stored) == 0 {
		t.Fatalf("feed was not stored in Fai: ok=%v err=%v", ok, err)
	}

	feed.Close()
	srv.syncCalendarFeeds(ctx)

	failed := fetch()
	if len(failed) != 1 || len(failed[0].Events) != 1 || failed[0].Error == "" {
		t.Fatalf("failed poll was not surfaced with cached events: %+v", failed)
	}
	if failed[0].LastSyncedAt == "" {
		t.Fatalf("failed poll did not report the last successful sync time: %+v", failed[0])
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("fresh cache refetched during a failed poll (%d hits)", got)
	}

	if err := st.SetCalendarFeed(ctx, source.ID, stored, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	offline := fetch()
	if len(offline) != 1 || len(offline[0].Events) != 1 || offline[0].Error == "" {
		t.Fatalf("stale cache was not served with a sync error when the feed was unreachable: %+v", offline)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("closed feed still counted %d hits", got)
	}
}
