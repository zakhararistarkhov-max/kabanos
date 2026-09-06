package meds

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func intp(n int) *int { return &n }

func TestBuildViewCourseDayAndStatus(t *testing.T) {
	cases := []struct {
		name      string
		start     string
		today     string
		duration  *int
		wantDay   int
		wantState string
	}{
		{"first day", "2026-09-06", "2026-09-06", intp(30), 1, "active"},
		{"third day", "2026-09-04", "2026-09-06", intp(30), 3, "active"},
		{"not started", "2026-09-10", "2026-09-06", intp(30), 0, "upcoming"},
		{"finished", "2026-08-01", "2026-09-06", intp(10), 37, "finished"},
		{"open ended", "2026-08-01", "2026-09-06", nil, 37, "active"},
		{"last day active", "2026-09-01", "2026-09-06", intp(6), 6, "active"},
	}
	for _, c := range cases {
		p := Progress{Medication: Medication{StartDate: d(c.start), DurationDays: c.duration}}
		v := buildView(p, d(c.today))
		if v.CourseDay != c.wantDay {
			t.Errorf("%s: courseDay=%d want %d", c.name, v.CourseDay, c.wantDay)
		}
		if v.Status != c.wantState {
			t.Errorf("%s: status=%q want %q", c.name, v.Status, c.wantState)
		}
	}
}

func TestDaysBetweenIgnoresTimezone(t *testing.T) {
	msk := time.FixedZone("MSK", 3*3600)
	a := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
	b := time.Date(2026, 9, 6, 1, 0, 0, 0, msk)
	if got := daysBetween(a, b); got != 2 {
		t.Fatalf("daysBetween=%d want 2", got)
	}
}
