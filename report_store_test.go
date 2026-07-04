package summarizer

import (
	"slices"
	"strings"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
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

func TestValidateCIDReportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		report      *shared.CIDReport
		expectedCID string
		wantErr     string
	}{
		{
			name: "matching cid",
			report: &shared.CIDReport{
				CID: "0f53593ceae34995af8fd295c18f1e25",
			},
			expectedCID: "0f53593ceae34995af8fd295c18f1e25",
		},
		{
			name: "mismatched cid",
			report: &shared.CIDReport{
				CID: "different-cid",
			},
			expectedCID: "0f53593ceae34995af8fd295c18f1e25",
			wantErr:     `CID mismatch: object key requested "0f53593ceae34995af8fd295c18f1e25" but report payload has "different-cid"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateCIDReportID(tt.report, tt.expectedCID)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateCIDReportID() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("validateCIDReportID() error = nil, want mismatch error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateCIDReportID() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateReportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cid     string
		wantErr string
	}{
		{
			name: "valid cid",
			cid:  "0f53593ceae34995af8fd295c18f1e25",
		},
		{
			name:    "empty cid",
			cid:     "",
			wantErr: "cid is empty",
		},
		{
			name:    "nested path not allowed",
			cid:     "team/abc123",
			wantErr: `cid "team/abc123" must not contain '/'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateReportID(tt.cid)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateReportID(%q) error = %v, want nil", tt.cid, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateReportID(%q) error = nil, want %q", tt.cid, tt.wantErr)
			}
			if got := err.Error(); got != tt.wantErr {
				t.Fatalf("validateReportID(%q) error = %q, want %q", tt.cid, got, tt.wantErr)
			}
		})
	}
}
