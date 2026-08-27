package store

import "testing"

func strPtr(s string) *string { return &s }

func TestSlotSortLess_OrdersByDateThenStartTimeThenDuration(t *testing.T) {
	cases := []struct {
		name                   string
		aStarts, bStarts       string
		aEnds, bEnds           *string
		wantALessB, wantBLessA bool
	}{
		{
			name:    "earlier date sorts first",
			aStarts: "2026-09-01", bStarts: "2026-09-02",
			wantALessB: true, wantBLessA: false,
		},
		{
			name:    "same date, earlier start time sorts first",
			aStarts: "2026-09-01T08:00", bStarts: "2026-09-01T09:00",
			aEnds: strPtr("2026-09-01T09:00"), bEnds: strPtr("2026-09-01T10:00"),
			wantALessB: true, wantBLessA: false,
		},
		{
			name:    "same date and start time, shorter duration sorts first",
			aStarts: "2026-09-01T08:00", bStarts: "2026-09-01T08:00",
			aEnds: strPtr("2026-09-01T08:30"), bEnds: strPtr("2026-09-01T10:00"),
			wantALessB: true, wantBLessA: false,
		},
		{
			name:    "identical slots are not less than each other either way",
			aStarts: "2026-09-01T08:00", bStarts: "2026-09-01T08:00",
			aEnds: strPtr("2026-09-01T09:00"), bEnds: strPtr("2026-09-01T09:00"),
			wantALessB: false, wantBLessA: false,
		},
		{
			name:    "all-day slot (no time) sorts before a specific-time slot on the same date",
			aStarts: "2026-09-01", bStarts: "2026-09-01T00:01",
			bEnds:      strPtr("2026-09-01T01:00"),
			wantALessB: true, wantBLessA: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := slotSortLess(tc.aStarts, tc.aEnds, tc.bStarts, tc.bEnds); got != tc.wantALessB {
				t.Fatalf("slotSortLess(a, b) = %v, want %v", got, tc.wantALessB)
			}
			if got := slotSortLess(tc.bStarts, tc.bEnds, tc.aStarts, tc.aEnds); got != tc.wantBLessA {
				t.Fatalf("slotSortLess(b, a) = %v, want %v", got, tc.wantBLessA)
			}
		})
	}
}

func TestSlotDurationMinutes(t *testing.T) {
	if got := slotDurationMinutes("2026-09-01", nil); got != 0 {
		t.Fatalf("expected all-day slot duration 0, got %d", got)
	}
	if got := slotDurationMinutes("2026-09-01T08:00", strPtr("2026-09-01T10:00")); got != 120 {
		t.Fatalf("expected 120 minutes, got %d", got)
	}
	if got := slotDurationMinutes("2026-09-01T23:30", strPtr("2026-09-01T00:30")); got != 60 {
		t.Fatalf("expected wrap-past-midnight duration of 60 minutes, got %d", got)
	}
}
