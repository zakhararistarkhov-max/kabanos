package training

import "strings"

// Option is a machine key paired with a Russian display label, mirroring the
// nutrition package's ActivityType so the frontend can render selects directly.
type Option struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Enumerations shown in the UI. Keys must match the CHECK constraints in the
// migration.
var (
	Categories = []Option{
		{"strength", "Силовое"},
		{"cardio", "Кардио"},
		{"mobility", "Мобильность / растяжка"},
	}
	Difficulties = []Option{
		{"easy", "Лёгкий"},
		{"medium", "Средний"},
		{"hard", "Тяжёлый"},
	}
	JointImpacts = []Option{
		{"low", "Низкая нагрузка на суставы"},
		{"medium", "Средняя нагрузка на суставы"},
		{"high", "Высокая нагрузка на суставы"},
	}
	// Equipment is a suggested list; users may also add their own tags.
	Equipment = []Option{
		{"none", "Без инвентаря"},
		{"dumbbells", "Гантели"},
		{"barbell", "Штанга"},
		{"kettlebell", "Гиря"},
		{"pullup_bar", "Турник"},
		{"dip_bars", "Брусья"},
		{"bench", "Скамья"},
		{"resistance_band", "Резинка"},
		{"mat", "Коврик"},
		{"machine", "Тренажёр"},
		{"jump_rope", "Скакалка"},
		{"box", "Тумба"},
		{"trx", "Петли TRX"},
	}
	// Muscles is a suggested list for the muscle-group tags.
	Muscles = []Option{
		{"chest", "Грудь"},
		{"back", "Спина"},
		{"shoulders", "Плечи"},
		{"biceps", "Бицепс"},
		{"triceps", "Трицепс"},
		{"legs", "Ноги"},
		{"glutes", "Ягодицы"},
		{"core", "Пресс / кор"},
		{"full_body", "Всё тело"},
	}
)

func validCategory(v string) bool   { return hasKey(Categories, v) }
func validDifficulty(v string) bool { return hasKey(Difficulties, v) }
func validJoint(v string) bool      { return hasKey(JointImpacts, v) }

func hasKey(opts []Option, v string) bool {
	for _, o := range opts {
		if o.Key == v {
			return true
		}
	}
	return false
}

// normalizeTags trims, lower-cases, de-duplicates and caps a tag list so free
// user input stays well-formed.
func normalizeTags(in []string, max int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || len(t) > 40 || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
		if len(out) >= max {
			break
		}
	}
	return out
}
