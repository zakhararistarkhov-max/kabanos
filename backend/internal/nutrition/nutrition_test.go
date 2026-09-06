package nutrition

import "testing"

func TestBurnedKcal(t *testing.T) {
	// 7.5 MET cycling, 80 kg, 60 min → 7.5*80*1 = 600 kcal.
	if got := BurnedKcal(7.5, 80, 60); got != 600 {
		t.Fatalf("BurnedKcal = %v, want 600", got)
	}
	// 30 min at 9.8 MET, 70 kg → 9.8*70*0.5 = 343.
	if got := BurnedKcal(9.8, 70, 30); got != 343 {
		t.Fatalf("BurnedKcal = %v, want 343", got)
	}
}

func TestScaleMacros(t *testing.T) {
	d := &Dish{KcalPer100: 250, ProteinPer100: 10, FatPer100: 8, CarbsPer100: 30}
	m := scaleMacros(d, 150.0/100.0) // 150 g
	if m.Kcal != 375 {
		t.Fatalf("kcal = %v, want 375", m.Kcal)
	}
	if m.Protein != 15 || m.Fat != 12 || m.Carbs != 45 {
		t.Fatalf("macros = %+v, want P15 F12 C45", m)
	}
}

func TestResolveGrams(t *testing.T) {
	serving := 200.0
	dishWithServing := &Dish{ServingGrams: &serving}
	dishNoServing := &Dish{}

	g := func(f float64) *float64 { return &f }

	cases := []struct {
		name     string
		dish     *Dish
		grams    *float64
		servings *float64
		want     float64
		wantErr  bool
	}{
		{"explicit grams", dishNoServing, g(120), nil, 120, false},
		{"servings with serving size", dishWithServing, nil, g(2), 400, false},
		{"servings without serving size", dishNoServing, nil, g(2), 0, true},
		{"default to one serving", dishWithServing, nil, nil, 200, false},
		{"default to 100g", dishNoServing, nil, nil, 100, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveGrams(tc.dish, tc.grams, tc.servings)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("grams = %v, want %v", got, tc.want)
			}
		})
	}
}
