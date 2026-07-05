package domain

import "testing"

func TestCIDReportValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		report  *CIDReport
		wantErr string
	}{
		{
			name: "valid report",
			report: &CIDReport{
				CID:     "abc123",
				NumAIDs: 10,
				Platforms: []PlatformSummary{
					{Name: "Windows 11", NumAIDs: 10},
				},
			},
		},
		{
			name: "missing cid",
			report: &CIDReport{
				NumAIDs: 10,
			},
			wantErr: "cid is empty",
		},
		{
			name: "invalid platform name",
			report: &CIDReport{
				CID:     "abc123",
				NumAIDs: 10,
				Platforms: []PlatformSummary{
					{Name: "", NumAIDs: 10},
				},
			},
			wantErr: "platforms[0].name is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.report.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", tt.wantErr)
			}
			if got := err.Error(); got != tt.wantErr {
				t.Fatalf("Validate() error = %q, want %q", got, tt.wantErr)
			}
		})
	}
}

func TestCloudAuditReportValidate(t *testing.T) {
	t.Parallel()

	report := &CloudAuditReport{}
	if err := report.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want empty audit error")
	}
}
