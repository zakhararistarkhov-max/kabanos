package reminders

import (
	"context"

	"github.com/google/uuid"
)

type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Reminder, error) {
	return s.repo.ListByUser(ctx, userID)
}
func (s *Service) ListByHabit(ctx context.Context, userID, habitID uuid.UUID) ([]Reminder, error) {
	return s.repo.ListByHabit(ctx, userID, habitID)
}
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (*Reminder, error) {
	return s.repo.Create(ctx, userID, in)
}
func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Reminder, error) {
	return s.repo.Update(ctx, id, userID, in)
}
func (s *Service) SetEnabled(ctx context.Context, id, userID uuid.UUID, enabled bool) error {
	return s.repo.SetEnabled(ctx, id, userID, enabled)
}
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}
