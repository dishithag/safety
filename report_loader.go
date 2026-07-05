package summarizer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

// LoadCIDReport reads a per-CID audit report fixture from disk.
func LoadCIDReport(path string) (*shared.CIDReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("load CID report: open %s: %w", path, err)
	}
	defer file.Close()

	return loadCIDReportReader(file)
}

// LoadCloudAuditReport reads the cloud-wide rollup audit report fixture.
func LoadCloudAuditReport(path string) (*shared.CloudAuditReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("load cloud audit report: open %s: %w", path, err)
	}
	defer file.Close()

	return loadCloudAuditReportReader(file)
}

func loadCIDReportReader(reader io.Reader) (*shared.CIDReport, error) {
	var report shared.CIDReport
	if err := loadJSONReader(reader, &report); err != nil {
		return nil, fmt.Errorf("load CID report: %w", err)
	}
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("load CID report: validate payload: %w", err)
	}
	return &report, nil
}

func loadCloudAuditReportReader(reader io.Reader) (*shared.CloudAuditReport, error) {
	var report shared.CloudAuditReport
	if err := loadJSONReader(reader, &report); err != nil {
		return nil, fmt.Errorf("load cloud audit report: %w", err)
	}
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("load cloud audit report: validate payload: %w", err)
	}
	return &report, nil
}

func loadJSONReader(reader io.Reader, target any) error {
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}
