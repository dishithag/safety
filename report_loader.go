package summarizer

import (
	"encoding/json"
	"fmt"
	"os"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

// LoadCIDReport reads a per-CID audit report fixture from disk.
func LoadCIDReport(path string) (*shared.CIDReport, error) {
	var report shared.CIDReport
	if err := loadJSON(path, &report); err != nil {
		return nil, fmt.Errorf("load CID report: %w", err)
	}
	return &report, nil
}

// LoadCloudAuditReport reads the cloud-wide rollup audit report fixture.
func LoadCloudAuditReport(path string) (*shared.CloudAuditReport, error) {
	var report shared.CloudAuditReport
	if err := loadJSON(path, &report); err != nil {
		return nil, fmt.Errorf("load cloud audit report: %w", err)
	}
	return &report, nil
}

func loadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return nil
}
