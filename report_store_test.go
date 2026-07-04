package summarizer

import (
	"slices"
	"testing"
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
