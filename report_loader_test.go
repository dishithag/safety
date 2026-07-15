package summarizer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestLoadCIDReport(t *testing.T) {
	report, err := LoadCIDReport("../../testdata/sample_audit_reports/cids/0f53593ceae34995af8fd295c18f1e25.json")
	if err != nil {
		t.Fatalf("LoadCIDReport() error = %v", err)
	}

	if report.CID != "0f53593ceae34995af8fd295c18f1e25" {
		t.Fatalf("CID = %q, want %q", report.CID, "0f53593ceae34995af8fd295c18f1e25")
	}
	if report.NumAIDs != 13490 {
		t.Fatalf("NumAIDs = %d, want %d", report.NumAIDs, 13490)
	}
	if len(report.Platforms) != 3 {
		t.Fatalf("len(Platforms) = %d, want %d", len(report.Platforms), 3)
	}

	first := report.Platforms[0]
	if first.Name != "Windows 11" {
		t.Fatalf("Platforms[0].Name = %q, want %q", first.Name, "Windows 11")
	}
	if _, ok := first.Compliance["secure_boot_enabled"]; !ok {
		t.Fatalf("Platforms[0].Compliance missing key %q", "secure_boot_enabled")
	}
}

func TestLoadCIDReportReaderWithHash(t *testing.T) {
	t.Parallel()

	const raw = `{"cid":"cid-123","num_aids":7,"average_overall_score":91,"average_os_score":89,"platforms":[]}`

	loaded, err := loadCIDReportReaderWithHash(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("loadCIDReportReaderWithHash() error = %v", err)
	}

	if loaded.Report.CID != "cid-123" {
		t.Fatalf("CID = %q, want %q", loaded.Report.CID, "cid-123")
	}
	if got, want := loaded.SourceSHA256, expectedSHA256([]byte(raw)); got != want {
		t.Fatalf("SourceSHA256 = %q, want %q", got, want)
	}
}

func TestLoadCIDReportSourceHashMatchesFixtureBytes(t *testing.T) {
	t.Parallel()

	const path = "../../testdata/sample_audit_reports/cids/0f53593ceae34995af8fd295c18f1e25.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	loaded, err := loadCIDReportReaderWithHash(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("loadCIDReportReaderWithHash() error = %v", err)
	}

	if got, want := loaded.SourceSHA256, expectedSHA256(raw); got != want {
		t.Fatalf("SourceSHA256 = %q, want %q", got, want)
	}
}

func TestLoadCloudAuditReport(t *testing.T) {
	report, err := LoadCloudAuditReport("../../testdata/sample_audit_reports/cloud_audit.json")
	if err != nil {
		t.Fatalf("LoadCloudAuditReport() error = %v", err)
	}

	if len(report.Audit) != 15 {
		t.Fatalf("len(Audit) = %d, want %d", len(report.Audit), 15)
	}

	first := report.Audit[0]
	if first.CID != "00000000000000000000000000000009c" {
		t.Fatalf("Audit[0].CID = %q, want %q", first.CID, "00000000000000000000000000000009c")
	}
	if len(first.Platforms) != 2 {
		t.Fatalf("len(Audit[0].Platforms) = %d, want %d", len(first.Platforms), 2)
	}
	if first.Platforms[0].Name != "Windows 10" {
		t.Fatalf("Audit[0].Platforms[0].Name = %q, want %q", first.Platforms[0].Name, "Windows 10")
	}
}

func expectedSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
