package httpapi

import "time"

// ResolveLocation prefers the IANA zone so DST transitions are correct, then
// falls back to a fixed offset in minutes east of UTC, then UTC.
func ResolveLocation(tz string, offsetMinutes *int) *time.Location {
	if tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	if offsetMinutes != nil && *offsetMinutes >= -14*60 && *offsetMinutes <= 14*60 {
		return time.FixedZone("local", *offsetMinutes*60)
	}
	return time.UTC
}
