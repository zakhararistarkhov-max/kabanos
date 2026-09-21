package decisions

import (
	"context"

	"github.com/google/uuid"
)

type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Decision, error) {
	return s.repo.List(ctx, userID)
}
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (*Decision, error) {
	return s.repo.Create(ctx, userID, in)
}
func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Decision, error) {
	return s.repo.Update(ctx, id, userID, in)
}
func (s *Service) Review(ctx context.Context, id, userID uuid.UUID, in ReviewInput) (*Decision, error) {
	return s.repo.Review(ctx, id, userID, in)
}
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}
