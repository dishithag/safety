package summarizer

import (
	"strings"
	"testing"
	"time"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestSummaryObjectKey(t *testing.T) {
	t.Parallel()

	const cid = "0f53593ceae34995af8fd295c18f1e25"
	if got, want := SummaryObjectKey(cid), "summary/cids/0f53593ceae34995af8fd295c18f1e25.md"; got != want {
		t.Fatalf("SummaryObjectKey(%q) = %q, want %q", cid, got, want)
	}
}

func TestCurrentSummaryProfile(t *testing.T) {
	t.Parallel()

	profile := CurrentSummaryProfile(&Config{
		NarrativeProvider: "genaihub",
		GenAIHubModel:     "claude-example",
	})

	if profile.Version != SummaryVersion {
		t.Fatalf("Version = %q, want %q", profile.Version, SummaryVersion)
	}
	if profile.NarrativeProvider != "genaihub" {
		t.Fatalf("NarrativeProvider = %q, want %q", profile.NarrativeProvider, "genaihub")
	}
	if profile.Model != "claude-example" {
		t.Fatalf("Model = %q, want %q", profile.Model, "claude-example")
	}
}

func TestSummaryMetadataMatches(t *testing.T) {
	t.Parallel()

	profile := SummaryProfile{
		Version:           SummaryVersion,
		NarrativeProvider: "placeholder",
	}
	metadata := NewSummaryMetadata("abc123", profile, time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC))

	if !metadata.Matches("abc123", profile) {
		t.Fatal("metadata should match the same source hash and profile")
	}
	if metadata.Matches("different", profile) {
		t.Fatal("metadata should not match a different source hash")
	}
	changedProfile := profile
	changedProfile.Version = "different"
	if metadata.Matches("abc123", changedProfile) {
		t.Fatal("metadata should not match a different summary version")
	}
}

func TestFallbackSummaryProfileNeverMatchesGenAIProfile(t *testing.T) {
	t.Parallel()

	genAIProfile := SummaryProfile{
		Version:           SummaryVersion,
		NarrativeProvider: "genaihub",
		Model:             "claude-example",
	}
	fallbackProfile := FallbackSummaryProfile(genAIProfile)
	if fallbackProfile.NarrativeProvider != FallbackNarrativeProvider {
		t.Fatalf("fallback provider = %q, want %q", fallbackProfile.NarrativeProvider, FallbackNarrativeProvider)
	}
	if fallbackProfile.Model != "" {
		t.Fatalf("fallback model = %q, want empty", fallbackProfile.Model)
	}

	metadata := NewSummaryMetadata("abc123", fallbackProfile, time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC))
	if metadata.Matches("abc123", genAIProfile) {
		t.Fatal("fallback metadata must not satisfy the requested GenAI profile")
	}
	if !metadata.Matches("abc123", fallbackProfile) {
		t.Fatal("fallback metadata should describe the fallback object that was written")
	}
}

func TestRenderPlaceholderSummary(t *testing.T) {
	t.Parallel()

	report := &shared.CIDReport{
		CID:                 "abc123",
		NumAIDs:             42,
		AverageOverallScore: 91.5,
		AverageOSScore:      88.0,
		Platforms: []shared.PlatformSummary{
			{
				Name:                "Windows 11",
				NumAIDs:             21,
				AverageOverallScore: 93.2,
				AverageOSScore:      89.1,
			},
		},
	}

	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	summary, err := RenderPlaceholderSummary(analysis)
	if err != nil {
		t.Fatalf("RenderPlaceholderSummary returned error: %v", err)
	}
	for _, want := range []string{
		"# Zero Trust Assessment Report",
		"**CID:** `abc123`",
		"**Reported devices:** **42**",
		"## High-Level Overview",
		"## 1. Windows 11",
		"## Recommended Next Steps",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q\nfull summary:\n%s", want, summary)
		}
	}
}
