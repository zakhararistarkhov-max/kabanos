package weight

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/timex"
	"github.com/kabanos/backend/internal/user"
)

// Service composes weight data with the user's height (from their profile) to
// derive BMI.
type Service struct {
	repo  *Repo
	users *user.Repo
}

func NewService(repo *Repo, users *user.Repo) *Service {
	return &Service{repo: repo, users: users}
}

func (s *Service) SetGoal(ctx context.Context, userID uuid.UUID, targetKg float64) (*Goal, error) {
	return s.repo.UpsertGoal(ctx, userID, targetKg)
}

// AddEntry upserts the measurement for measuredOn (defaults to today in tz).
func (s *Service) AddEntry(ctx context.Context, userID uuid.UUID, weightKg float64, note, measuredOn, tz string) (*Entry, error) {
	loc := timex.Location(tz)
	if measuredOn == "" {
		measuredOn = timex.Today(loc)
	}
	if _, err := timex.ParseDate(measuredOn, loc); err != nil {
		return nil, err
	}
	return s.repo.UpsertEntry(ctx, userID, weightKg, note, measuredOn, time.Now())
}

func (s *Service) DeleteEntry(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteEntry(ctx, userID, id)
}

// Summary builds the weight dashboard: latest weight, target, BMI (if height is
// known) and the recent series.
func (s *Service) Summary(ctx context.Context, userID uuid.UUID, limit int) (*Summary, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	series, err := s.repo.List(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	sum := &Summary{HeightCm: u.HeightCm, Series: series}

	if g, err := s.repo.GetGoal(ctx, userID); err == nil {
		sum.TargetKg = &g.TargetKg
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}

	if latest, err := s.repo.Latest(ctx, userID); err == nil {
		w := latest.WeightKg
		sum.LatestKg = &w
		sum.BMI = BMI(w, u.HeightCm)
		sum.BMICategory = BMICategory(sum.BMI)
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}

	return sum, nil
}
