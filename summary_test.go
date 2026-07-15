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
	changedProfile.Version = "v2"
	if metadata.Matches("abc123", changedProfile) {
		t.Fatal("metadata should not match a different summary version")
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

	summary := RenderPlaceholderSummary(report)
	for _, want := range []string{
		"## Zero Trust Audit Narrative",
		"### Executive Summary",
		"CID `abc123` covers 42 aids across 1 platforms.",
		"### Platform Highlights",
		"- Windows 11: 21 aids, overall 93.20, OS 89.10.",
		"### Demo Note",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q\nfull summary:\n%s", want, summary)
		}
	}
}
