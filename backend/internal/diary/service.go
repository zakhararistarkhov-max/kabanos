package diary

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/storage"
)

var (
	ErrUnsupportedMedia = errors.New("unsupported media type")
	ErrStorageDisabled  = errors.New("media storage is disabled")
	ErrInvalidAttachment = errors.New("invalid attachment")
)

// mediaExt maps accepted content types (images and video) to file extensions.
var mediaExt = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/gif":       ".gif",
	"video/mp4":       ".mp4",
	"video/webm":      ".webm",
	"video/quicktime": ".mov",
}

func kindOf(contentType string) string {
	if strings.HasPrefix(contentType, "video/") {
		return "video"
	}
	return "image"
}

type Service struct {
	repo  *Repo
	store *storage.Storage
}

func NewService(repo *Repo, store *storage.Storage) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) Day(ctx context.Context, userID uuid.UUID, date string) ([]Entry, error) {
	return s.repo.ListByDate(ctx, userID, date)
}

func (s *Service) Dates(ctx context.Context, userID uuid.UUID, from, to string) ([]DateCount, error) {
	return s.repo.DatesWithCounts(ctx, userID, from, to)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, date, body string, atts []Attachment) (*Entry, error) {
	norm, err := normalizeAttachments(atts)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, userID, date, body, norm)
}

func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, body string, atts []Attachment) (*Entry, error) {
	norm, err := normalizeAttachments(atts)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, userID, body, norm)
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}

// normalizeAttachments validates each attachment and derives its kind from the
// content type.
func normalizeAttachments(atts []Attachment) ([]Attachment, error) {
	out := make([]Attachment, 0, len(atts))
	for _, a := range atts {
		if strings.TrimSpace(a.Key) == "" {
			return nil, ErrInvalidAttachment
		}
		if _, ok := mediaExt[a.ContentType]; !ok {
			return nil, ErrUnsupportedMedia
		}
		out = append(out, Attachment{Key: a.Key, ContentType: a.ContentType, Kind: kindOf(a.ContentType)})
	}
	return out, nil
}

// PrepareUpload validates the media type, mints a key and returns a presigned
// PUT URL plus the derived kind.
func (s *Service) PrepareUpload(ctx context.Context, contentType string) (uploadURL, key, kind string, err error) {
	if !s.store.Enabled() {
		return "", "", "", ErrStorageDisabled
	}
	ext, ok := mediaExt[contentType]
	if !ok {
		return "", "", "", ErrUnsupportedMedia
	}
	key = storage.NewKey("diary", ext)
	uploadURL, err = s.store.PresignPut(ctx, key, contentType, 15*time.Minute)
	return uploadURL, key, kindOf(contentType), err
}

// MediaURL returns a short-lived GET URL for a stored object (nil-safe).
func (s *Service) MediaURL(ctx context.Context, key string) *string {
	if key == "" || !s.store.Enabled() {
		return nil
	}
	u, err := s.store.PresignGet(ctx, key, 2*time.Hour)
	if err != nil {
		return nil
	}
	return &u
}
