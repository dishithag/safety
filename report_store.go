package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

var (
	// ErrCIDReportNotFound indicates a requested per-CID report object is missing.
	ErrCIDReportNotFound = errors.New("cid report not found")
	// ErrCloudAuditReportNotFound indicates the cloud-wide rollup object is missing.
	ErrCloudAuditReportNotFound = errors.New("cloud audit report not found")
	// ErrSummaryMetadataNotFound indicates a requested summary metadata object is missing.
	ErrSummaryMetadataNotFound = errors.New("summary metadata not found")
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
	seen := make(map[string]struct{})
	for object := range objects {
		if object.Err != nil {
			return nil, fmt.Errorf("list report objects: %w", object.Err)
		}
		id, ok := cidFromReportObjectKey(object.Key)
		if !ok {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids, nil
}

// LoadCIDReportFromStore fetches, hashes, and parses a per-CID audit report from object storage.
func (s *ReportStore) LoadCIDReportFromStore(ctx context.Context, cid string) (*LoadedCIDReport, error) {
	reader, err := s.fetchObject(ctx, reportObjectKey(cid), ErrCIDReportNotFound)
	if err != nil {
		return nil, fmt.Errorf("load CID report from store: %w", err)
	}
	defer reader.Close()

	report, err := loadCIDReportReaderWithHash(reader)
	if err != nil {
		return nil, fmt.Errorf("load CID report from store: %w", err)
	}

	return report, nil
}

// LoadCloudAuditReportFromStore fetches and parses the cloud rollup report from object storage.
func (s *ReportStore) LoadCloudAuditReportFromStore(ctx context.Context) (*shared.CloudAuditReport, error) {
	reader, err := s.fetchObject(ctx, cloudAuditObjectKey, ErrCloudAuditReportNotFound)
	if err != nil {
		return nil, fmt.Errorf("load cloud audit report from store: %w", err)
	}
	defer reader.Close()

	report, err := loadCloudAuditReportReader(reader)
	if err != nil {
		return nil, fmt.Errorf("load cloud audit report from store: %w", err)
	}
	return report, nil
}

// WriteSummary stores a Markdown narrative for the given CID at the agreed
// summary object-store key, followed by a metadata sidecar used for freshness checks.
// Re-writing the same keys is safe.
func (s *ReportStore) WriteSummary(ctx context.Context, cid string, markdown string, metadata SummaryMetadata) error {
	key := SummaryObjectKey(cid)
	reader := bytes.NewReader([]byte(markdown))

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, int64(reader.Len()), minio.PutObjectOptions{
		ContentType: "text/markdown; charset=utf-8",
	})
	if err != nil {
		return fmt.Errorf("write summary %s: %w", key, err)
	}

	metadataKey := SummaryMetadataObjectKey(cid)
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary metadata %s: %w", metadataKey, err)
	}
	metadataBytes = append(metadataBytes, '\n')
	metadataReader := bytes.NewReader(metadataBytes)

	_, err = s.client.PutObject(ctx, s.bucket, metadataKey, metadataReader, int64(metadataReader.Len()), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("write summary metadata %s: %w", metadataKey, err)
	}

	return nil
}

// LoadSummaryMetadata fetches the metadata sidecar for a CID summary.
func (s *ReportStore) LoadSummaryMetadata(ctx context.Context, cid string) (SummaryMetadata, bool, error) {
	key := SummaryMetadataObjectKey(cid)

	reader, err := s.fetchObject(ctx, key, ErrSummaryMetadataNotFound)
	if err != nil {
		if errors.Is(err, ErrSummaryMetadataNotFound) {
			return SummaryMetadata{}, false, nil
		}
		return SummaryMetadata{}, false, fmt.Errorf("load summary metadata from store: %w", err)
	}
	defer reader.Close()

	var metadata SummaryMetadata
	if err := json.NewDecoder(reader).Decode(&metadata); err != nil {
		return SummaryMetadata{}, false, fmt.Errorf("decode summary metadata %s: %w", key, err)
	}

	return metadata, true, nil
}

func (s *ReportStore) fetchObject(ctx context.Context, key string, notFoundErr error) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		if isMissingObjectError(err) {
			return nil, fmt.Errorf("%w: %s", notFoundErr, key)
		}
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}

	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		if isMissingObjectError(err) {
			return nil, fmt.Errorf("%w: %s", notFoundErr, key)
		}
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

func isMissingObjectError(err error) bool {
	resp := minio.ToErrorResponse(err)
	return resp.Code == "NoSuchKey" || resp.Code == "NoSuchObject"
}
