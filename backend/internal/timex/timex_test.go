package timex

import "testing"

func TestDayRangeIsTimezoneAware(t *testing.T) {
	loc := Location("Europe/Moscow") // UTC+3
	from, to, err := DayRange("2026-09-06", loc)
	if err != nil {
		t.Fatalf("DayRange: %v", err)
	}
	// Midnight Moscow on 2026-09-06 is 21:00 UTC on 2026-09-05.
	if got := from.UTC().Format("2006-01-02T15:04"); got != "2026-09-05T21:00" {
		t.Fatalf("from = %s, want 2026-09-05T21:00 UTC", got)
	}
	if got := to.Sub(from).Hours(); got != 24 {
		t.Fatalf("day length = %v hours, want 24", got)
	}
}

func TestLocationFallsBackToUTC(t *testing.T) {
	if Location("Not/AZone").String() != "UTC" {
		t.Fatal("invalid timezone should fall back to UTC")
	}
	if Location("").String() != "UTC" {
		t.Fatal("empty timezone should be UTC")
	}
}

func TestRangeBoundsInclusive(t *testing.T) {
	loc := Location("UTC")
	from, to, err := RangeBounds("2026-01-01", "2026-01-03", loc)
	if err != nil {
		t.Fatalf("RangeBounds: %v", err)
	}
	if days := int(to.Sub(from).Hours() / 24); days != 3 {
		t.Fatalf("range covers %d days, want 3 (inclusive)", days)
	}
}
