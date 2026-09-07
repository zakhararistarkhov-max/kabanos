package nutrition

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/storage"
	"github.com/kabanos/backend/internal/timex"
)

// Domain errors surfaced to the handler.
var (
	ErrServingUnknown = errors.New("dish has no serving size; specify grams")
	ErrNoWeight       = errors.New("body weight required to compute burned calories; add a weight entry or provide kcal")
	ErrInvalidActivity = errors.New("provide either kcal, or an activity type/MET with duration")
	ErrUnsupportedMedia = errors.New("unsupported image type")
	ErrStorageDisabled  = errors.New("image storage is disabled")
	ErrIngredientNotFound = errors.New("ingredient dish not found")
)

// imageExtensions maps accepted content types to file extensions.
var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// WeightProvider supplies the user's latest body weight for MET calculations.
// Implemented in main over the weight repo, keeping this package decoupled.
type WeightProvider interface {
	LatestKg(ctx context.Context, userID uuid.UUID) (kg float64, ok bool, err error)
}

type Service struct {
	dishes  *DishRepo
	log     *LogRepo
	store   *storage.Storage
	weights WeightProvider
}

func NewService(dishes *DishRepo, log *LogRepo, store *storage.Storage, weights WeightProvider) *Service {
	return &Service{dishes: dishes, log: log, store: store, weights: weights}
}

// defaultGoal is a reasonable placeholder split (25/30/45 P/F/C at 2000 kcal).
func defaultGoal() Goal { return Goal{Kcal: 2000, Protein: 125, Fat: 67, Carbs: 225} }

// --- goals ---

func (s *Service) GetGoalOrDefault(ctx context.Context, userID uuid.UUID) (Goal, error) {
	g, err := s.log.GetGoal(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return defaultGoal(), nil
		}
		return Goal{}, err
	}
	return *g, nil
}

func (s *Service) SetGoal(ctx context.Context, userID uuid.UUID, g Goal) (Goal, error) {
	out, err := s.log.UpsertGoal(ctx, userID, g)
	if err != nil {
		return Goal{}, err
	}
	return *out, nil
}

// --- dishes (thin passthroughs; ownership enforced in repo) ---

func (s *Service) CreateDish(ctx context.Context, d *Dish, ingredients []IngredientInput) (*Dish, error) {
	return s.dishes.Create(ctx, d, ingredients)
}
func (s *Service) UpdateDish(ctx context.Context, d *Dish, ingredients []IngredientInput, owner uuid.UUID) (*Dish, error) {
	return s.dishes.Update(ctx, d, ingredients, owner)
}
func (s *Service) DeleteDish(ctx context.Context, id, owner uuid.UUID) error {
	return s.dishes.Delete(ctx, id, owner)
}
func (s *Service) PublishDish(ctx context.Context, id, owner uuid.UUID, public bool) error {
	return s.dishes.SetPublic(ctx, id, owner, public)
}
func (s *Service) GetDish(ctx context.Context, id, viewer uuid.UUID) (*Dish, error) {
	return s.dishes.GetByID(ctx, id, viewer)
}
func (s *Service) ListDishes(ctx context.Context, p ListParams) ([]Dish, int, error) {
	return s.dishes.List(ctx, p)
}
func (s *Service) RateDish(ctx context.Context, dishID, userID uuid.UUID, rating int) error {
	return s.dishes.SetRating(ctx, dishID, userID, rating)
}
func (s *Service) UnrateDish(ctx context.Context, dishID, userID uuid.UUID) error {
	return s.dishes.DeleteRating(ctx, dishID, userID)
}
func (s *Service) AddComment(ctx context.Context, dishID, userID uuid.UUID, body string) (*Comment, error) {
	return s.dishes.AddComment(ctx, dishID, userID, body)
}
func (s *Service) ListComments(ctx context.Context, dishID uuid.UUID, limit int) ([]Comment, error) {
	return s.dishes.ListComments(ctx, dishID, limit)
}
func (s *Service) DeleteComment(ctx context.Context, commentID, userID uuid.UUID) error {
	return s.dishes.DeleteComment(ctx, commentID, userID)
}
func (s *Service) Favorite(ctx context.Context, userID, dishID uuid.UUID) error {
	return s.dishes.AddFavorite(ctx, userID, dishID)
}
func (s *Service) Unfavorite(ctx context.Context, userID, dishID uuid.UUID) error {
	return s.dishes.RemoveFavorite(ctx, userID, dishID)
}

// --- diet log ---

// AddDishToDiet computes the macro snapshot from grams (or servings) and logs it.
func (s *Service) AddDishToDiet(ctx context.Context, userID, dishID uuid.UUID, grams, servings *float64, meal *string, consumedAt *time.Time) (*DietEntry, error) {
	dish, err := s.dishes.GetByID(ctx, dishID, userID)
	if err != nil {
		return nil, err
	}

	g, err := resolveGrams(dish, grams, servings)
	if err != nil {
		return nil, err
	}
	factor := g / 100.0
	entry := DietEntry{
		DishID:     &dish.ID,
		Name:       dish.Name,
		Grams:      &g,
		Meal:       meal,
		Source:     "dish",
		Macros:     scaleMacros(dish, factor),
		ConsumedAt: at(consumedAt),
	}
	return s.log.AddDietEntry(ctx, userID, entry)
}

func resolveGrams(dish *Dish, grams, servings *float64) (float64, error) {
	switch {
	case grams != nil:
		return *grams, nil
	case servings != nil:
		if dish.ServingGrams == nil {
			return 0, ErrServingUnknown
		}
		return *servings * *dish.ServingGrams, nil
	case dish.ServingGrams != nil:
		return *dish.ServingGrams, nil // default to one serving
	default:
		return 100, nil // default to 100 g
	}
}

func scaleMacros(dish *Dish, factor float64) Macros {
	return Macros{
		Kcal:    math.Round(dish.KcalPer100 * factor),
		Protein: round1(dish.ProteinPer100 * factor),
		Fat:     round1(dish.FatPer100 * factor),
		Carbs:   round1(dish.CarbsPer100 * factor),
	}
}

// AddManualEntry logs food entered by hand.
func (s *Service) AddManualEntry(ctx context.Context, userID uuid.UUID, name string, m Macros, meal *string, consumedAt *time.Time) (*DietEntry, error) {
	entry := DietEntry{
		Name:       name,
		Meal:       meal,
		Source:     "manual",
		Macros:     Macros{Kcal: math.Round(m.Kcal), Protein: round1(m.Protein), Fat: round1(m.Fat), Carbs: round1(m.Carbs)},
		ConsumedAt: at(consumedAt),
	}
	return s.log.AddDietEntry(ctx, userID, entry)
}

func (s *Service) DeleteDietEntry(ctx context.Context, userID, id uuid.UUID) error {
	return s.log.DeleteDietEntry(ctx, userID, id)
}

// --- activities ---

// AddActivity logs a calorie burn. If kcal is provided it is used directly;
// otherwise kcal is derived from MET × body weight × duration.
func (s *Service) AddActivity(ctx context.Context, userID uuid.UUID, typ string, met *float64, durationMin *int, kcal *float64, performedAt *time.Time) (*Activity, error) {
	a := Activity{Type: typ, PerformedAt: at(performedAt), DurationMin: durationMin}

	if kcal != nil {
		a.Kcal = math.Round(*kcal)
		a.Source = "manual"
		a.MET = met
		return s.log.AddActivity(ctx, userID, a)
	}

	// MET path: need a MET value (explicit or from the preset) and a duration.
	metVal := 0.0
	if met != nil {
		metVal = *met
	} else if v, ok := metFor(typ); ok {
		metVal = v
	} else {
		return nil, ErrInvalidActivity
	}
	if durationMin == nil || *durationMin <= 0 {
		return nil, ErrInvalidActivity
	}

	weightKg, ok, err := s.weights.LatestKg(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNoWeight
	}

	a.Kcal = BurnedKcal(metVal, weightKg, *durationMin)
	a.MET = &metVal
	a.Source = "met"
	return s.log.AddActivity(ctx, userID, a)
}

func (s *Service) DeleteActivity(ctx context.Context, userID, id uuid.UUID) error {
	return s.log.DeleteActivity(ctx, userID, id)
}

// --- day summary & history ---

func (s *Service) DaySummary(ctx context.Context, userID uuid.UUID, date, tz string) (*DaySummary, error) {
	loc := timex.Location(tz)
	if date == "" {
		date = timex.Today(loc)
	}
	from, to, err := timex.DayRange(date, loc)
	if err != nil {
		return nil, err
	}

	goal, err := s.GetGoalOrDefault(ctx, userID)
	if err != nil {
		return nil, err
	}
	entries, err := s.log.ListDietEntries(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	activities, err := s.log.ListActivities(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	var consumed Macros
	for _, e := range entries {
		consumed.Kcal += e.Macros.Kcal
		consumed.Protein += e.Macros.Protein
		consumed.Fat += e.Macros.Fat
		consumed.Carbs += e.Macros.Carbs
	}
	var burned float64
	for _, a := range activities {
		burned += a.Kcal
	}
	net := consumed.Kcal - burned
	return &DaySummary{
		Date:          date,
		Goal:          goal,
		Consumed:      roundMacros(consumed),
		BurnedKcal:    burned,
		NetKcal:       net,
		RemainingKcal: float64(goal.Kcal) - net,
		Entries:       entries,
		Activities:    activities,
	}, nil
}

func (s *Service) History(ctx context.Context, userID uuid.UUID, fromDate, toDate, tz string) ([]DayTotals, Goal, error) {
	loc := timex.Location(tz)
	from, to, err := timex.RangeBounds(fromDate, toDate, loc)
	if err != nil {
		return nil, Goal{}, err
	}
	byDay, err := s.log.DailyTotals(ctx, userID, from, to, loc.String())
	if err != nil {
		return nil, Goal{}, err
	}
	goal, err := s.GetGoalOrDefault(ctx, userID)
	if err != nil {
		return nil, Goal{}, err
	}

	out := []DayTotals{}
	for cur := from; cur.Before(to); cur = cur.AddDate(0, 0, 1) {
		key := cur.Format("2006-01-02")
		d := byDay[key]
		d.Date = key
		out = append(out, roundDay(d))
	}
	return out, goal, nil
}

// --- images ---

// PrepareImageUpload validates the media type, generates an object key and
// returns a presigned URL the browser can PUT the file to.
func (s *Service) PrepareImageUpload(ctx context.Context, contentType string) (uploadURL, key string, err error) {
	if !s.store.Enabled() {
		return "", "", ErrStorageDisabled
	}
	ext, ok := imageExtensions[contentType]
	if !ok {
		return "", "", ErrUnsupportedMedia
	}
	key = storage.NewImageKey(ext)
	uploadURL, err = s.store.PresignPut(ctx, key, contentType, 10*time.Minute)
	return uploadURL, key, err
}

// ImageURL returns a short-lived GET URL for a stored image key (nil-safe).
func (s *Service) ImageURL(ctx context.Context, key *string) *string {
	if key == nil || *key == "" || !s.store.Enabled() {
		return nil
	}
	u, err := s.store.PresignGet(ctx, *key, time.Hour)
	if err != nil {
		return nil
	}
	return &u
}

func (s *Service) StorageEnabled() bool { return s.store.Enabled() }

// --- helpers ---

func at(t *time.Time) time.Time {
	if t != nil {
		return *t
	}
	return time.Now()
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

func roundMacros(m Macros) Macros {
	return Macros{Kcal: math.Round(m.Kcal), Protein: round1(m.Protein), Fat: round1(m.Fat), Carbs: round1(m.Carbs)}
}

func roundDay(d DayTotals) DayTotals {
	d.Kcal = math.Round(d.Kcal)
	d.Protein = round1(d.Protein)
	d.Fat = round1(d.Fat)
	d.Carbs = round1(d.Carbs)
	d.BurnedKcal = math.Round(d.BurnedKcal)
	return d
}
