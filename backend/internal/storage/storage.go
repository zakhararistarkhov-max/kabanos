// Package storage wraps S3-compatible object storage (MinIO) for user-uploaded
// media (dish photos). Uploads and downloads happen directly between the browser
// and the store via presigned URLs, so large files never pass through the API.
//
// Two clients are used: `admin` talks to the internal endpoint (bucket setup);
// `public` signs URLs for the browser-facing endpoint so the signature matches
// the host the browser actually connects to.
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/kabanos/backend/internal/config"
)

var ErrDisabled = errors.New("object storage is disabled")

type Storage struct {
	admin   *minio.Client
	public  *minio.Client
	bucket  string
	enabled bool
}

// New builds the storage clients and ensures the bucket exists.
func New(ctx context.Context, cfg config.S3) (*Storage, error) {
	if !cfg.Enabled {
		return &Storage{enabled: false}, nil
	}
	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")

	admin, err := minio.New(cfg.Endpoint, &minio.Options{Creds: creds, Secure: cfg.UseSSL, Region: cfg.Region})
	if err != nil {
		return nil, fmt.Errorf("minio admin client: %w", err)
	}
	public, err := minio.New(cfg.PublicEndpoint, &minio.Options{Creds: creds, Secure: cfg.UseSSL, Region: cfg.Region})
	if err != nil {
		return nil, fmt.Errorf("minio public client: %w", err)
	}

	// Retry the bucket check/create so the API tolerates MinIO still starting up
	// (e.g. right after `docker compose up`).
	if err := ensureBucket(ctx, admin, cfg.Bucket, cfg.Region); err != nil {
		return nil, err
	}
	return &Storage{admin: admin, public: public, bucket: cfg.Bucket, enabled: true}, nil
}

func ensureBucket(ctx context.Context, admin *minio.Client, bucket, region string) error {
	const attempts = 10
	var lastErr error
	for i := 0; i < attempts; i++ {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		exists, err := admin.BucketExists(callCtx, bucket)
		if err == nil {
			if !exists {
				if mkErr := admin.MakeBucket(callCtx, bucket, minio.MakeBucketOptions{Region: region}); mkErr != nil {
					// A concurrent creator may have won the race; treat "exists"
					// as success on the next check.
					lastErr = mkErr
					cancel()
					if ok, _ := admin.BucketExists(ctx, bucket); ok {
						return nil
					}
					time.Sleep(2 * time.Second)
					continue
				}
			}
			cancel()
			return nil
		}
		cancel()
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("ensure bucket after %d attempts: %w", attempts, lastErr)
}

func (s *Storage) Enabled() bool { return s.enabled }

// NewImageKey returns a fresh object key for a dish image with the given
// extension (validated by the caller).
func NewImageKey(ext string) string {
	return "dishes/" + uuid.NewString() + ext
}

// PresignPut returns a URL the browser can PUT the file to directly. The
// content type is not bound into the signature, so the browser may send any
// image type it declared; validation happens before the key is issued.
func (s *Storage) PresignPut(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	if !s.enabled {
		return "", ErrDisabled
	}
	_ = contentType
	u, err := s.public.PresignedPutObject(ctx, s.bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PresignGet returns a short-lived URL the browser can GET the file from.
func (s *Storage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if !s.enabled || key == "" {
		return "", ErrDisabled
	}
	u, err := s.public.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
