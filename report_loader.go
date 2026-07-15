package summarizer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

// LoadedCIDReport contains a parsed CID report and a hash of its source JSON.
type LoadedCIDReport struct {
	Report       *shared.CIDReport
	SourceSHA256 string
}

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
	loaded, err := loadCIDReportReaderWithHash(reader)
	if err != nil {
		return nil, err
	}
	return loaded.Report, nil
}

func loadCIDReportReaderWithHash(reader io.Reader) (*LoadedCIDReport, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("load CID report: read JSON: %w", err)
	}

	var report shared.CIDReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, fmt.Errorf("load CID report: %w", err)
	}
	return &LoadedCIDReport{
		Report:       &report,
		SourceSHA256: sourceSHA256(raw),
	}, nil
}

func loadCloudAuditReportReader(reader io.Reader) (*shared.CloudAuditReport, error) {
	var report shared.CloudAuditReport
	if err := loadJSONReader(reader, &report); err != nil {
		return nil, fmt.Errorf("load cloud audit report: %w", err)
	}
	return &report, nil
}

func loadJSONReader(reader io.Reader, target any) error {
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

func sourceSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
