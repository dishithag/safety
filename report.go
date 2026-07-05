package domain

import "fmt"

// ComplianceMap stores per-signal compliance percentages keyed by signal name.
type ComplianceMap map[string]float64

// PlatformSummary captures a platform-level score breakdown within a report.
type PlatformSummary struct {
	NumAIDs                  int           `json:"num_aids"`
	Name                     string        `json:"name"`
	AverageOverallScore      float64       `json:"average_overall_score"`
	AverageOSScore           float64       `json:"average_os_score"`
	AverageSensorConfigScore float64       `json:"average_sensor_config_score,omitempty"`
	Compliance               ComplianceMap `json:"compliance,omitempty"`
}

// CIDReport models a per-customer audit report.
type CIDReport struct {
	CID                      string            `json:"cid"`
	NumAIDs                  int               `json:"num_aids"`
	AverageOverallScore      float64           `json:"average_overall_score"`
	AverageOSScore           float64           `json:"average_os_score"`
	AverageSensorConfigScore float64           `json:"average_sensor_config_score,omitempty"`
	Platforms                []PlatformSummary `json:"platforms"`
}

// CloudAuditReport models the cloud-wide rollup report, which is a list of
// per-CID summaries under the top-level "audit" key.
type CloudAuditReport struct {
	Audit []CIDReport `json:"audit"`
}

// Validate checks that a per-CID report has the minimum structure the
// summarizer expects before any downstream processing.
func (r *CIDReport) Validate() error {
	if r == nil {
		return fmt.Errorf("report is nil")
	}
	if r.CID == "" {
		return fmt.Errorf("cid is empty")
	}
	if r.NumAIDs < 0 {
		return fmt.Errorf("num_aids must be non-negative: %d", r.NumAIDs)
	}
	for i := range r.Platforms {
		if err := r.Platforms[i].validate(i); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks that the cloud rollup contains at least one CID report and
// that each embedded report is structurally valid.
func (r *CloudAuditReport) Validate() error {
	if r == nil {
		return fmt.Errorf("report is nil")
	}
	if len(r.Audit) == 0 {
		return fmt.Errorf("audit is empty")
	}
	for i := range r.Audit {
		if err := r.Audit[i].Validate(); err != nil {
			return fmt.Errorf("audit[%d]: %w", i, err)
		}
	}
	return nil
}

func (p *PlatformSummary) validate(index int) error {
	if p.Name == "" {
		return fmt.Errorf("platforms[%d].name is empty", index)
	}
	if p.NumAIDs < 0 {
		return fmt.Errorf("platforms[%d].num_aids must be non-negative: %d", index, p.NumAIDs)
	}
	return nil
}
