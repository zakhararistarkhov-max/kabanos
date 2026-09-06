package nutrition

import "math"

// ActivityType is a preset with a MET (Metabolic Equivalent of Task) value.
// Burned kcal ≈ MET × bodyWeightKg × hours.
type ActivityType struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	MET   float64 `json:"met"`
}

// ActivityTypes are the built-in presets offered in the UI. Values are typical
// mid-range METs from the Compendium of Physical Activities.
var ActivityTypes = []ActivityType{
	{"walking", "Ходьба", 3.5},
	{"brisk_walking", "Быстрая ходьба", 4.3},
	{"running", "Бег", 9.8},
	{"cycling", "Велосипед", 7.5},
	{"swimming", "Плавание", 8.0},
	{"strength", "Силовая тренировка", 6.0},
	{"hiit", "HIIT / интервальная", 8.0},
	{"yoga", "Йога", 3.0},
	{"elliptical", "Эллипс", 5.0},
	{"rowing", "Гребля", 7.0},
	{"dancing", "Танцы", 5.0},
	{"hiking", "Поход / трекинг", 6.0},
}

func metFor(key string) (float64, bool) {
	for _, a := range ActivityTypes {
		if a.Key == key {
			return a.MET, true
		}
	}
	return 0, false
}

// BurnedKcal computes calories burned from MET, body weight and minutes,
// rounded to a whole kcal.
func BurnedKcal(met, weightKg float64, minutes int) float64 {
	kcal := met * weightKg * (float64(minutes) / 60.0)
	return math.Round(kcal)
}
