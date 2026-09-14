package openfoodfacts

import "testing"

func TestNormalizeKcalAndFields(t *testing.T) {
	var r offResponse
	r.Status = 1
	r.Product.ProductName = "Chicken breast"
	r.Product.Brands = "BrandA, BrandB"
	r.Product.ServingQty = "150"
	r.Product.Nutriments = map[string]any{
		"energy-kcal_100g":   float64(165),
		"proteins_100g":      float64(31),
		"fat_100g":           "3.6", // OFF sometimes sends strings
		"carbohydrates_100g": float64(0),
	}

	p := normalize("0000000000000", r)
	if p.Name != "Chicken breast" {
		t.Errorf("name = %q", p.Name)
	}
	if p.Brand != "BrandA" {
		t.Errorf("brand = %q, want first brand only", p.Brand)
	}
	if p.KcalPer100 != 165 || p.ProteinPer100 != 31 || p.FatPer100 != 3.6 {
		t.Errorf("macros = %+v", p)
	}
	if p.ServingGrams == nil || *p.ServingGrams != 150 {
		t.Errorf("serving = %v", p.ServingGrams)
	}
}

func TestKcalFromKilojoules(t *testing.T) {
	// No kcal field → convert from kJ (energy_100g is kJ).
	got := kcalFromNutriments(map[string]any{"energy_100g": float64(1000)})
	if got < 238 || got > 240 { // 1000 / 4.184 ≈ 239
		t.Errorf("kcal from kJ = %v, want ~239", got)
	}
	if kcalFromNutriments(map[string]any{"energy-kcal_100g": float64(240)}) != 240 {
		t.Errorf("explicit kcal should win")
	}
}

func TestNormalizeFallsBackName(t *testing.T) {
	var r offResponse
	r.Status = 1
	r.Product.GenericName = "" // nothing usable
	p := normalize("123", r)
	if p.Name == "" {
		t.Error("name should fall back to a non-empty placeholder")
	}
}
