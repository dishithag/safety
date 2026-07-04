package summarizer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

const (
	reportsPrefix       = "reports/cids/"
	reportObjectSuffix  = ".json"
	cloudAuditObjectKey = "reports/cloud_audit.json"
)

// ReportStore reads audit report inputs from an S3-compatible object store.
type ReportStore struct {
	bucket string
	client *minio.Client
}

// NewReportStore constructs a MinIO-backed report store from summarizer config.
func NewReportStore(cfg *Config) (*ReportStore, error) {
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

	return &ReportStore{
		bucket: cfg.S3Bucket,
		client: client,
	}, nil
}

// ListCIDReportIDs returns the discovered per-CID report ids from object storage.
func (s *ReportStore) ListCIDReportIDs(ctx context.Context) ([]string, error) {
	objects := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    reportsPrefix,
		Recursive: true,
	})

	var ids []string
	for object := range objects {
		if object.Err != nil {
			return nil, fmt.Errorf("list report objects: %w", object.Err)
		}
		id, ok := cidFromReportObjectKey(object.Key)
		if !ok {
			continue
		}
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids, nil
}

// LoadCIDReportFromStore fetches and parses a per-CID audit report from object storage.
func (s *ReportStore) LoadCIDReportFromStore(ctx context.Context, cid string) (*shared.CIDReport, error) {
	reader, err := s.fetchObject(ctx, reportObjectKey(cid))
	if err != nil {
		return nil, fmt.Errorf("load CID report from store: %w", err)
	}
	defer reader.Close()

	return loadCIDReportReader(reader)
}

// LoadCloudAuditReportFromStore fetches and parses the cloud rollup report from object storage.
func (s *ReportStore) LoadCloudAuditReportFromStore(ctx context.Context) (*shared.CloudAuditReport, error) {
	reader, err := s.fetchObject(ctx, cloudAuditObjectKey)
	if err != nil {
		return nil, fmt.Errorf("load cloud audit report from store: %w", err)
	}
	defer reader.Close()

	return loadCloudAuditReportReader(reader)
}

func (s *ReportStore) fetchObject(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}

	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("stat object %s: %w", key, err)
	}

	return obj, nil
}

func reportObjectKey(cid string) string {
	return reportsPrefix + cid + reportObjectSuffix
}

func cidFromReportObjectKey(key string) (string, bool) {
	if !strings.HasPrefix(key, reportsPrefix) || !strings.HasSuffix(key, reportObjectSuffix) {
		return "", false
	}

	id := strings.TrimSuffix(strings.TrimPrefix(key, reportsPrefix), reportObjectSuffix)
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}

	return id, true
}
