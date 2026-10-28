package httpapi

import "testing"

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
