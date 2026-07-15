package summarizer

import (
	"strings"
	"time"
)

const (
	SummaryVersion      = "v2"
	summaryPrefix       = "summary/cids/"
	summaryObjectSuffix = ".md"
)

// SummaryProfile identifies the generation inputs that affect narrative output.
type SummaryProfile struct {
	Version           string
	NarrativeProvider string
	Model             string
}

// SummaryMetadata records the source report and generation profile stored on a summary object.
type SummaryMetadata struct {
	SourceSHA256      string
	SummaryVersion    string
	NarrativeProvider string
	Model             string
	GeneratedAt       time.Time
}

// CurrentSummaryProfile returns the active narrative generation profile.
func CurrentSummaryProfile(cfg *Config) SummaryProfile {
	provider := "placeholder"
	model := ""
	if cfg != nil {
		provider = effectiveNarrativeProvider(cfg.NarrativeProvider)
		if provider == "genaihub" {
			model = strings.TrimSpace(cfg.GenAIHubModel)
		}
	}

	return SummaryProfile{
		Version:           SummaryVersion,
		NarrativeProvider: provider,
		Model:             model,
	}
}

// NewSummaryMetadata builds the metadata written on each summary object.
func NewSummaryMetadata(sourceSHA256 string, profile SummaryProfile, generatedAt time.Time) SummaryMetadata {
	return SummaryMetadata{
		SourceSHA256:      sourceSHA256,
		SummaryVersion:    profile.Version,
		NarrativeProvider: profile.NarrativeProvider,
		Model:             profile.Model,
		GeneratedAt:       generatedAt.UTC(),
	}
}

// Matches reports whether metadata is current for the report content and generation profile.
func (m SummaryMetadata) Matches(sourceSHA256 string, profile SummaryProfile) bool {
	return m.SourceSHA256 == sourceSHA256 &&
		m.SummaryVersion == profile.Version &&
		m.NarrativeProvider == profile.NarrativeProvider &&
		m.Model == profile.Model
}

// SummaryObjectKey returns the object-store key for a CID narrative summary.
func SummaryObjectKey(cid string) string {
	return summaryPrefix + cid + summaryObjectSuffix
}

func effectiveNarrativeProvider(provider string) string {
	if strings.TrimSpace(provider) == "" {
		return "placeholder"
	}
	return strings.TrimSpace(provider)
}
