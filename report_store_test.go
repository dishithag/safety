package summarizer

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

func TestCIDFromReportObjectKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
		ok   bool
	}{
		{
			name: "valid report object key",
			key:  "reports/cids/0f53593ceae34995af8fd295c18f1e25.json",
			want: "0f53593ceae34995af8fd295c18f1e25",
			ok:   true,
		},
		{
			name: "wrong prefix",
			key:  "summary/cids/0f53593ceae34995af8fd295c18f1e25.json",
			ok:   false,
		},
		{
			name: "wrong extension",
			key:  "reports/cids/0f53593ceae34995af8fd295c18f1e25.md",
			ok:   false,
		},
		{
			name: "nested path not allowed",
			key:  "reports/cids/team/0f53593ceae34995af8fd295c18f1e25.json",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := cidFromReportObjectKey(tt.key)
			if ok != tt.ok {
				t.Fatalf("cidFromReportObjectKey(%q) ok = %v, want %v", tt.key, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("cidFromReportObjectKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestReportObjectKey(t *testing.T) {
	t.Parallel()

	const cid = "0f53593ceae34995af8fd295c18f1e25"
	if got, want := reportObjectKey(cid), "reports/cids/0f53593ceae34995af8fd295c18f1e25.json"; got != want {
		t.Fatalf("reportObjectKey(%q) = %q, want %q", cid, got, want)
	}
}

func TestListOrderHelperExpectation(t *testing.T) {
	t.Parallel()

	got := []string{"c", "a", "b"}
	slices.Sort(got)
	if want := []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Fatalf("sorted ids = %v, want %v", got, want)
	}
}

func TestSummaryObjectMetadataRoundTrip(t *testing.T) {
	t.Parallel()

	genAI := SummaryMetadata{
		SourceSHA256:      "abc123",
		SummaryVersion:    "v2",
		NarrativeProvider: "genaihub",
		Model:             "claude-example",
		GeneratedAt:       time.Date(2026, 7, 14, 12, 34, 56, 123, time.UTC),
	}
	placeholder := SummaryMetadata{
		SourceSHA256:      "def456",
		SummaryVersion:    "v1",
		NarrativeProvider: "placeholder",
		GeneratedAt:       time.Date(2026, 7, 14, 12, 34, 56, 0, time.UTC),
	}
	genAIMetadata := summaryObjectUserMetadata(genAI)

	tests := []struct {
		name       string
		objectInfo minio.ObjectInfo
		want       SummaryMetadata
	}{
		{
			name: "minio user metadata",
			objectInfo: minio.ObjectInfo{
				UserMetadata: minio.StringMap(genAIMetadata),
			},
			want: genAI,
		},
		{
			name: "s3 metadata headers",
			objectInfo: minio.ObjectInfo{
				Metadata: metadataHeaders(genAIMetadata),
			},
			want: genAI,
		},
		{
			name: "placeholder metadata without model",
			objectInfo: minio.ObjectInfo{
				UserMetadata: minio.StringMap(summaryObjectUserMetadata(placeholder)),
			},
			want: placeholder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, exists, err := summaryMetadataFromObjectInfo(tt.objectInfo)
			if err != nil {
				t.Fatalf("summaryMetadataFromObjectInfo() error = %v", err)
			}
			if !exists {
				t.Fatal("summaryMetadataFromObjectInfo() exists = false, want true")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("summaryMetadataFromObjectInfo() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSummaryMetadataFromObjectInfoMissing(t *testing.T) {
	t.Parallel()

	metadata, exists, err := summaryMetadataFromObjectInfo(minio.ObjectInfo{})
	if err != nil {
		t.Fatalf("summaryMetadataFromObjectInfo() error = %v", err)
	}
	if exists {
		t.Fatal("summaryMetadataFromObjectInfo() exists = true, want false")
	}
	if metadata != (SummaryMetadata{}) {
		t.Fatalf("summaryMetadataFromObjectInfo() = %+v, want zero value", metadata)
	}
}

func TestSummaryMetadataFromObjectInfoRejectsIncompleteMetadata(t *testing.T) {
	t.Parallel()

	_, _, err := summaryMetadataFromObjectInfo(minio.ObjectInfo{
		UserMetadata: minio.StringMap{
			metadataSourceSHA256Key: "abc123",
		},
	})
	if err == nil {
		t.Fatal("summaryMetadataFromObjectInfo() error = nil, want incomplete metadata error")
	}
}

func metadataHeaders(values map[string]string) http.Header {
	headers := make(http.Header, len(values))
	for key, value := range values {
		headers.Set(userMetadataHeaderPrefix+key, value)
	}
	return headers
}
