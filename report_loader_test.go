package summarizer

import (
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

func TestLoadCIDReportReaderRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	_, err := loadCIDReportReader(strings.NewReader(`{"cid":"","num_aids":1,"platforms":[]}`))
	if err == nil {
		t.Fatal("loadCIDReportReader() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "validate payload: cid is empty") {
		t.Fatalf("loadCIDReportReader() error = %q, want cid validation error", err.Error())
	}
}

func TestLoadCloudAuditReportReaderRejectsEmptyAudit(t *testing.T) {
	t.Parallel()

	_, err := loadCloudAuditReportReader(strings.NewReader(`{"audit":[]}`))
	if err == nil {
		t.Fatal("loadCloudAuditReportReader() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "validate payload: audit is empty") {
		t.Fatalf("loadCloudAuditReportReader() error = %q, want empty audit validation error", err.Error())
	}
}
