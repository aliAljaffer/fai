package httpapi

import (
	"testing"
	"time"
)

func TestResolveLocation(t *testing.T) {
	three, huge := 180, 99999
	tests := []struct {
		name   string
		tz     string
		offset *int
		want   string
		secs   int
	}{
		{"valid zone", "Asia/Riyadh", nil, "Asia/Riyadh", 3 * 3600},
		{"valid zone wins over offset", "Asia/Riyadh", &huge, "Asia/Riyadh", 3 * 3600},
		{"invalid zone uses offset", "Not/AZone", &three, "local", 3 * 3600},
		{"offset only", "", &three, "local", 3 * 3600},
		{"offset out of range", "", &huge, "UTC", 0},
		{"both missing", "", nil, "UTC", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loc := ResolveLocation(tc.tz, tc.offset)
			if loc.String() != tc.want {
				t.Fatalf("name = %q, want %q", loc.String(), tc.want)
			}
			if _, off := time.Date(2026, 1, 1, 12, 0, 0, 0, loc).Zone(); off != tc.secs {
				t.Fatalf("offset = %d, want %d", off, tc.secs)
			}
		})
	}
}

func TestResolveLocationDST(t *testing.T) {
	loc := ResolveLocation("America/New_York", nil)
	before := time.Date(2026, 3, 7, 12, 0, 0, 0, loc)
	after := time.Date(2026, 3, 9, 12, 0, 0, 0, loc)
	if _, off := before.Zone(); off != -5*3600 {
		t.Fatalf("before offset = %d", off)
	}
	if _, off := after.Zone(); off != -4*3600 {
		t.Fatalf("after offset = %d", off)
	}
}
