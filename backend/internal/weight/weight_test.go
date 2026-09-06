package weight

import "testing"

func ptr(f float64) *float64 { return &f }

func TestBMI(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		height   *float64
		wantBMI  *float64
		wantCat  string
	}{
		{"normal", 70, ptr(175), ptr(22.9), "normal"},
		{"underweight", 45, ptr(170), ptr(15.6), "underweight"},
		{"overweight", 85, ptr(175), ptr(27.8), "overweight"},
		{"obese", 110, ptr(175), ptr(35.9), "obese"},
		{"no height", 70, nil, nil, ""},
		{"zero height", 70, ptr(0), nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BMI(tt.weight, tt.height)
			if (got == nil) != (tt.wantBMI == nil) {
				t.Fatalf("BMI nil mismatch: got %v want %v", got, tt.wantBMI)
			}
			if got != nil && *got != *tt.wantBMI {
				t.Fatalf("BMI = %v, want %v", *got, *tt.wantBMI)
			}
			if cat := BMICategory(got); cat != tt.wantCat {
				t.Fatalf("category = %q, want %q", cat, tt.wantCat)
			}
		})
	}
}
