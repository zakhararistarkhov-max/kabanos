package training

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/storage"
)

// Domain errors surfaced to the handler.
var (
	ErrUnsupportedMedia = errors.New("unsupported image type")
	ErrStorageDisabled  = errors.New("image storage is disabled")
	ErrUnknownExercise  = errors.New("one or more exercises do not exist")
	ErrEmptyWorkout     = errors.New("a workout needs at least one exercise")
)

var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type Service struct {
	ex    *ExerciseRepo
	wo    *WorkoutRepo
	store *storage.Storage
}

func NewService(ex *ExerciseRepo, wo *WorkoutRepo, store *storage.Storage) *Service {
	return &Service{ex: ex, wo: wo, store: store}
}

// --- exercises ---

// CreateExercise normalizes the tag lists then persists.
func (s *Service) CreateExercise(ctx context.Context, e *Exercise) (*Exercise, error) {
	e.Equipment = normalizeTags(e.Equipment, 12)
	e.Muscles = normalizeTags(e.Muscles, 12)
	return s.ex.Create(ctx, e)
}

func (s *Service) UpdateExercise(ctx context.Context, e *Exercise, owner uuid.UUID) (*Exercise, error) {
	e.Equipment = normalizeTags(e.Equipment, 12)
	e.Muscles = normalizeTags(e.Muscles, 12)
	return s.ex.Update(ctx, e, owner)
}

func (s *Service) DeleteExercise(ctx context.Context, id, owner uuid.UUID) error {
	return s.ex.Delete(ctx, id, owner)
}
func (s *Service) GetExercise(ctx context.Context, id, viewer uuid.UUID) (*Exercise, error) {
	return s.ex.GetByID(ctx, id, viewer)
}
func (s *Service) ListExercises(ctx context.Context, p ListParams) ([]Exercise, int, error) {
	return s.ex.List(ctx, p)
}
func (s *Service) RateExercise(ctx context.Context, id, user uuid.UUID, rating int) error {
	return s.ex.SetRating(ctx, id, user, rating)
}
func (s *Service) UnrateExercise(ctx context.Context, id, user uuid.UUID) error {
	return s.ex.DeleteRating(ctx, id, user)
}
func (s *Service) FavoriteExercise(ctx context.Context, user, id uuid.UUID) error {
	return s.ex.AddFavorite(ctx, user, id)
}
func (s *Service) UnfavoriteExercise(ctx context.Context, user, id uuid.UUID) error {
	return s.ex.RemoveFavorite(ctx, user, id)
}
func (s *Service) AddExerciseComment(ctx context.Context, id, user uuid.UUID, body string) (*Comment, error) {
	return s.ex.AddComment(ctx, id, user, body)
}
func (s *Service) ListExerciseComments(ctx context.Context, id uuid.UUID) ([]Comment, error) {
	return s.ex.ListComments(ctx, id, 0)
}
func (s *Service) DeleteExerciseComment(ctx context.Context, commentID, user uuid.UUID) error {
	return s.ex.DeleteComment(ctx, commentID, user)
}

// --- workouts ---

func (s *Service) CreateWorkout(ctx context.Context, w *Workout, items []ItemInput) (*Workout, error) {
	if err := s.validateItems(ctx, items); err != nil {
		return nil, err
	}
	return s.wo.Create(ctx, w, items)
}

func (s *Service) UpdateWorkout(ctx context.Context, w *Workout, items []ItemInput, owner uuid.UUID) (*Workout, error) {
	if err := s.validateItems(ctx, items); err != nil {
		return nil, err
	}
	return s.wo.Update(ctx, w, items, owner)
}

// validateItems ensures the workout has at least one item and every referenced
// exercise exists (any user's exercise may be used).
func (s *Service) validateItems(ctx context.Context, items []ItemInput) error {
	if len(items) == 0 {
		return ErrEmptyWorkout
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ExerciseID)
	}
	ok, err := s.ex.existAll(ctx, ids)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUnknownExercise
	}
	return nil
}

func (s *Service) DeleteWorkout(ctx context.Context, id, owner uuid.UUID) error {
	return s.wo.Delete(ctx, id, owner)
}
func (s *Service) GetWorkout(ctx context.Context, id, viewer uuid.UUID) (*Workout, error) {
	return s.wo.GetByID(ctx, id, viewer)
}
func (s *Service) ListWorkouts(ctx context.Context, p ListParams) ([]Workout, int, error) {
	return s.wo.List(ctx, p)
}
func (s *Service) WorkoutItemCounts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	return s.wo.ItemCounts(ctx, ids)
}
func (s *Service) RateWorkout(ctx context.Context, id, user uuid.UUID, rating int) error {
	return s.wo.SetRating(ctx, id, user, rating)
}
func (s *Service) UnrateWorkout(ctx context.Context, id, user uuid.UUID) error {
	return s.wo.DeleteRating(ctx, id, user)
}
func (s *Service) FavoriteWorkout(ctx context.Context, user, id uuid.UUID) error {
	return s.wo.AddFavorite(ctx, user, id)
}
func (s *Service) UnfavoriteWorkout(ctx context.Context, user, id uuid.UUID) error {
	return s.wo.RemoveFavorite(ctx, user, id)
}
func (s *Service) AddWorkoutComment(ctx context.Context, id, user uuid.UUID, body string) (*Comment, error) {
	return s.wo.AddComment(ctx, id, user, body)
}
func (s *Service) ListWorkoutComments(ctx context.Context, id uuid.UUID) ([]Comment, error) {
	return s.wo.ListComments(ctx, id, 0)
}
func (s *Service) DeleteWorkoutComment(ctx context.Context, commentID, user uuid.UUID) error {
	return s.wo.DeleteComment(ctx, commentID, user)
}

// --- media (shared) ---

// PrepareImageUpload validates the media type, mints an object key under the
// given prefix and returns a presigned PUT URL for the browser.
func (s *Service) PrepareImageUpload(ctx context.Context, prefix, contentType string) (uploadURL, key string, err error) {
	if !s.store.Enabled() {
		return "", "", ErrStorageDisabled
	}
	ext, ok := imageExtensions[contentType]
	if !ok {
		return "", "", ErrUnsupportedMedia
	}
	key = storage.NewKey(prefix, ext)
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
