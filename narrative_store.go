package repos

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi/domain"
)

// NarrativeStore reads stored narrative markdown from an S3-compatible object store.
type NarrativeStore struct {
	bucket string
	client *minio.Client
}

// NewNarrativeStore constructs a minio-backed narrative store from analyticsapi config.
func NewNarrativeStore(cfg *domain.Config) (*NarrativeStore, error) {
	u, err := url.Parse(cfg.S3Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse S3 endpoint: %w", err)
	}

	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: u.Scheme == "https",
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &NarrativeStore{
		bucket: cfg.S3Bucket,
		client: client,
	}, nil
}

// Get fetches a summary markdown object for the given CID.
func (s *NarrativeStore) Get(ctx context.Context, cid string) (string, error) {
	key := domain.NarrativeObjectKey(cid)

	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("get object %s: %w", key, err)
	}

	if _, err := obj.Stat(); err != nil {
		if resp := minio.ToErrorResponse(err); resp.Code == "NoSuchKey" || resp.Code == "NoSuchObject" {
			return "", fmt.Errorf("%w: %s", domain.ErrNarrativeNotFound, cid)
		}
		return "", fmt.Errorf("stat object %s: %w", key, err)
	}

	body, err := io.ReadAll(obj)
	if err != nil {
		return "", fmt.Errorf("read object %s: %w", key, err)
	}

	return string(body), nil
}
