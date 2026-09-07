package pressure

import "testing"

func TestCategory(t *testing.T) {
	cases := []struct {
		sys, dia int
		want     string
	}{
		{110, 70, "normal"},
		{119, 79, "normal"},
		{125, 78, "elevated"},
		{135, 85, "high1"},
		{129, 82, "high1"}, // diastolic drives stage 1
		{145, 92, "high2"},
		{160, 85, "high2"}, // systolic drives stage 2
		{185, 100, "crisis"},
		{170, 125, "crisis"}, // diastolic drives crisis
		{0, 0, ""},
	}
	for _, c := range cases {
		if got := Category(c.sys, c.dia); got != c.want {
			t.Errorf("Category(%d,%d)=%q want %q", c.sys, c.dia, got, c.want)
		}
	}
}

func TestAverage(t *testing.T) {
	p := func(n int) *int { return &n }
	series := []Entry{
		{Systolic: 120, Diastolic: 80, Pulse: p(60)},
		{Systolic: 130, Diastolic: 90, Pulse: nil}, // pulse skipped in avg
		{Systolic: 140, Diastolic: 100, Pulse: p(80)},
	}
	a := average(series)
	if a.Systolic == nil || *a.Systolic != 130 {
		t.Fatalf("systolic avg = %v want 130", a.Systolic)
	}
	if a.Diastolic == nil || *a.Diastolic != 90 {
		t.Fatalf("diastolic avg = %v want 90", a.Diastolic)
	}
	if a.Pulse == nil || *a.Pulse != 70 {
		t.Fatalf("pulse avg = %v want 70 (over 2 entries)", a.Pulse)
	}
	if empty := average(nil); empty.Systolic != nil {
		t.Fatalf("empty average should be nil")
	}
}
