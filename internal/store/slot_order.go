package store

import "strings"

// splitStartsAt splits a starts_at value into its date portion and, for
// date_time slots, its "HH:MM" time portion. All-day slots store a bare
// date with no "T" separator, so timePart is "" for them.
func splitStartsAt(startsAt string) (date, timePart string) {
	if idx := strings.IndexByte(startsAt, 'T'); idx != -1 {
		return startsAt[:idx], startsAt[idx+1:]
	}
	return startsAt, ""
}

// timeToMinutes converts an "HH:MM" string to minutes since midnight. An
// empty or malformed value (e.g. an all-day slot's absent time) is treated
// as 0 (midnight) — the natural sort position for a slot with no time.
func timeToMinutes(t string) int {
	if len(t) != 5 || t[2] != ':' {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}

// slotDurationMinutes returns a slot's duration in minutes, or 0 for an
// all-day slot (no ends_at).
func slotDurationMinutes(startsAt string, endsAt *string) int {
	if endsAt == nil {
		return 0
	}
	_, startTime := splitStartsAt(startsAt)
	_, endTime := splitStartsAt(*endsAt)
	d := timeToMinutes(endTime) - timeToMinutes(startTime)
	if d < 0 {
		d += 1440
	}
	return d
}

// slotSortLess orders slots by date, then start time, then duration
// (shortest first). Ties are left in whatever order the caller's stable
// sort presents them in.
func slotSortLess(aStartsAt string, aEndsAt *string, bStartsAt string, bEndsAt *string) bool {
	aDate, aTime := splitStartsAt(aStartsAt)
	bDate, bTime := splitStartsAt(bStartsAt)
	if aDate != bDate {
		return aDate < bDate
	}
	aMinutes := timeToMinutes(aTime)
	bMinutes := timeToMinutes(bTime)
	if aMinutes != bMinutes {
		return aMinutes < bMinutes
	}
	return slotDurationMinutes(aStartsAt, aEndsAt) < slotDurationMinutes(bStartsAt, bEndsAt)
}
