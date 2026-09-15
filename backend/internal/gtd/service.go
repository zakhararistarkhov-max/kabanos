package gtd

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/storage"
)

type Service struct {
	repo  *Repo
	store *storage.Storage
}

func NewService(repo *Repo, store *storage.Storage) *Service {
	return &Service{repo: repo, store: store}
}

// Image storage errors surfaced to the graph handler.
var (
	ErrUnsupportedMedia = errors.New("unsupported image type")
	ErrStorageDisabled  = errors.New("image storage is disabled")
)

var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// PrepareImageUpload returns a presigned PUT URL + key for a node photo.
func (s *Service) PrepareImageUpload(ctx context.Context, contentType string) (uploadURL, key string, err error) {
	if s.store == nil || !s.store.Enabled() {
		return "", "", ErrStorageDisabled
	}
	ext, ok := imageExtensions[contentType]
	if !ok {
		return "", "", ErrUnsupportedMedia
	}
	key = storage.NewKey("gtd", ext)
	uploadURL, err = s.store.PresignPut(ctx, key, contentType, 10*time.Minute)
	return uploadURL, key, err
}

// imageURLs resolves stored image keys to short-lived GET URLs.
func (s *Service) imageURLs(ctx context.Context, keys []string) []string {
	out := make([]string, 0, len(keys))
	if s.store == nil || !s.store.Enabled() {
		return out
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		if u, err := s.store.PresignGet(ctx, k, time.Hour); err == nil {
			out = append(out, u)
		}
	}
	return out
}

// --- projects ---

func (s *Service) CreateProject(ctx context.Context, userID uuid.UUID, in ProjectInput) (*Project, error) {
	return s.repo.CreateProject(ctx, userID, in)
}
func (s *Service) UpdateProject(ctx context.Context, id, userID uuid.UUID, in ProjectInput) (*Project, error) {
	return s.repo.UpdateProject(ctx, id, userID, in)
}
func (s *Service) DeleteProject(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.DeleteProject(ctx, id, userID)
}
func (s *Service) ListProjects(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	return s.repo.ListProjects(ctx, userID)
}

// --- items ---

func (s *Service) CreateItem(ctx context.Context, userID uuid.UUID, in ItemInput) (*Item, error) {
	return s.repo.CreateItem(ctx, userID, in)
}
func (s *Service) UpdateItem(ctx context.Context, id, userID uuid.UUID, in ItemInput) (*Item, error) {
	return s.repo.UpdateItem(ctx, id, userID, in)
}
func (s *Service) DeleteItem(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.DeleteItem(ctx, id, userID)
}
func (s *Service) SetDone(ctx context.Context, id, userID uuid.UUID, done bool) (*Item, error) {
	return s.repo.SetDone(ctx, id, userID, done)
}
func (s *Service) ListItems(ctx context.Context, userID uuid.UUID, f ItemFilter) ([]Item, error) {
	return s.repo.ListItems(ctx, userID, f)
}
func (s *Service) Contexts(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return s.repo.Contexts(ctx, userID)
}

// Review assembles the state the weekly review surfaces in one round of queries.
func (s *Service) Review(ctx context.Context, userID uuid.UUID) (*Review, error) {
	counts, err := s.repo.OpenCountsByBucket(ctx, userID)
	if err != nil {
		return nil, err
	}
	byCtx, err := s.repo.ContextCounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	stalled, err := s.repo.StalledProjects(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	overdue, err := s.repo.OverdueCalendar(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	upcoming, err := s.repo.UpcomingCalendarCount(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	completed, err := s.repo.CompletedSince(ctx, userID, now.AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	return &Review{
		InboxCount:        counts["inbox"],
		NextCount:         counts["next"],
		WaitingCount:      counts["waiting"],
		SomedayCount:      counts["someday"],
		CalendarUpcoming:  upcoming,
		ByContext:         byCtx,
		StalledProjects:   stalled,
		OverdueCalendar:   overdue,
		CompletedThisWeek: completed,
	}, nil
}
