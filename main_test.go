package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
	"go.crwd.dev/ce/zerotrust-analytics/internal/summarizer"
)

func TestServiceName(t *testing.T) {
	if serviceName == "" {
		t.Fatal("serviceName should not be empty")
	}
}

func TestRunProcessesAllReportsAndContinuesAfterPerCIDFailures(t *testing.T) {
	profile := summarizer.SummaryProfile{
		Version:           summarizer.SummaryVersion,
		NarrativeProvider: "placeholder",
	}
	store := &fakeReportStore{
		ids: []string{"current", "stale", "missing-metadata", "bad-load", "bad-metadata", "bad-summary", "bad-write"},
		reports: map[string]*summarizer.LoadedCIDReport{
			"current":          loadedReport("current", "hash-current"),
			"stale":            loadedReport("stale", "hash-stale"),
			"missing-metadata": loadedReport("missing-metadata", "hash-missing-metadata"),
			"bad-summary":      loadedReport("bad-summary", "hash-bad-summary"),
			"bad-write":        loadedReport("bad-write", "hash-bad-write"),
		},
		metadata: map[string]summarizer.SummaryMetadata{
			"current": summarizer.NewSummaryMetadata("hash-current", profile, testGeneratedAt()),
			"stale":   summarizer.NewSummaryMetadata("old-hash", profile, testGeneratedAt()),
		},
		loadErr: map[string]error{
			"bad-load": errors.New("load failed"),
		},
		metadataErr: map[string]error{
			"bad-metadata": errors.New("metadata failed"),
		},
		writeErr: map[string]error{
			"bad-write": errors.New("write failed"),
		},
	}
	generator := &fakeNarrativeGenerator{
		errByCID: map[string]error{
			"bad-summary": errors.New("summarize failed"),
		},
	}

	stats, err := run(context.Background(), discardLogger(), store, generator, profile)
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	want := runStats{Total: 7, Processed: 2, Skipped: 1, Failed: 4}
	if stats != want {
		t.Fatalf("stats = %+v, want %+v", stats, want)
	}

	assertStringSlices(t, store.loaded, []string{"current", "stale", "missing-metadata", "bad-load", "bad-metadata", "bad-summary", "bad-write"})
	assertStringSlices(t, store.metadataChecked, []string{"current", "stale", "missing-metadata", "bad-metadata", "bad-summary", "bad-write"})
	assertStringSlices(t, generator.summarized, []string{"stale", "missing-metadata", "bad-summary", "bad-write"})
	assertStringSlices(t, store.writeAttempts, []string{"stale", "missing-metadata", "bad-write"})

	if got, want := store.writtenMetadata["stale"].SourceSHA256, "hash-stale"; got != want {
		t.Fatalf("stale written SourceSHA256 = %q, want %q", got, want)
	}
	if got, want := store.writtenMetadata["missing-metadata"].SourceSHA256, "hash-missing-metadata"; got != want {
		t.Fatalf("missing-metadata written SourceSHA256 = %q, want %q", got, want)
	}
}

func TestRunRegeneratesWhenSummaryProfileChanges(t *testing.T) {
	oldProfile := summarizer.SummaryProfile{
		Version:           summarizer.SummaryVersion,
		NarrativeProvider: "placeholder",
	}
	newProfile := summarizer.SummaryProfile{
		Version:           summarizer.SummaryVersion,
		NarrativeProvider: "genaihub",
		Model:             "claude-example",
	}
	store := &fakeReportStore{
		ids: []string{"same-report"},
		reports: map[string]*summarizer.LoadedCIDReport{
			"same-report": loadedReport("same-report", "same-hash"),
		},
		metadata: map[string]summarizer.SummaryMetadata{
			"same-report": summarizer.NewSummaryMetadata("same-hash", oldProfile, testGeneratedAt()),
		},
	}
	generator := &fakeNarrativeGenerator{}

	stats, err := run(context.Background(), discardLogger(), store, generator, newProfile)
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}
	if want := (runStats{Total: 1, Processed: 1}); stats != want {
		t.Fatalf("stats = %+v, want %+v", stats, want)
	}
	assertStringSlices(t, generator.summarized, []string{"same-report"})
	if got, want := store.writtenMetadata["same-report"].NarrativeProvider, "genaihub"; got != want {
		t.Fatalf("written NarrativeProvider = %q, want %q", got, want)
	}
	if got, want := store.writtenMetadata["same-report"].Model, "claude-example"; got != want {
		t.Fatalf("written Model = %q, want %q", got, want)
	}
}

func TestRunReturnsListError(t *testing.T) {
	store := &fakeReportStore{listErr: errors.New("list failed")}

	stats, err := run(context.Background(), discardLogger(), store, &fakeNarrativeGenerator{}, summarizer.SummaryProfile{})
	if err == nil {
		t.Fatal("expected list error")
	}
	if stats != (runStats{}) {
		t.Fatalf("stats = %+v, want zero value", stats)
	}
}

func TestRunNoReports(t *testing.T) {
	stats, err := run(context.Background(), discardLogger(), &fakeReportStore{}, &fakeNarrativeGenerator{}, summarizer.SummaryProfile{})
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}
	if stats != (runStats{}) {
		t.Fatalf("stats = %+v, want zero value", stats)
	}
}

func TestRunWithOptionsLimitsConcurrentSummaries(t *testing.T) {
	store := &fakeReportStore{
		ids: []string{"one", "two", "three", "four"},
	}
	generator := &concurrencyTrackingGenerator{delay: 25 * time.Millisecond}

	stats, err := runWithOptions(
		context.Background(),
		discardLogger(),
		store,
		generator,
		summarizer.SummaryProfile{},
		runOptions{Concurrency: 2, GenerationTimeout: time.Second},
	)
	if err != nil {
		t.Fatalf("runWithOptions returned unexpected error: %v", err)
	}
	if want := (runStats{Total: 4, Processed: 4}); stats != want {
		t.Fatalf("stats = %+v, want %+v", stats, want)
	}
	if got, want := generator.maximumConcurrency(), 2; got != want {
		t.Fatalf("maximum concurrent summaries = %d, want %d", got, want)
	}
}

func TestRunWithOptionsTimesOutStalledSummary(t *testing.T) {
	store := &fakeReportStore{ids: []string{"stalled"}}
	started := time.Now()

	stats, err := runWithOptions(
		context.Background(),
		discardLogger(),
		store,
		blockingNarrativeGenerator{},
		summarizer.SummaryProfile{},
		runOptions{Concurrency: 1, GenerationTimeout: 20 * time.Millisecond},
	)
	if err != nil {
		t.Fatalf("runWithOptions returned unexpected error: %v", err)
	}
	if want := (runStats{Total: 1, Failed: 1}); stats != want {
		t.Fatalf("stats = %+v, want %+v", stats, want)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("stalled summary took %s, want less than 1s", elapsed)
	}
}

type fakeReportStore struct {
	mu              sync.Mutex
	ids             []string
	listErr         error
	reports         map[string]*summarizer.LoadedCIDReport
	metadata        map[string]summarizer.SummaryMetadata
	metadataErr     map[string]error
	loadErr         map[string]error
	writeErr        map[string]error
	loaded          []string
	metadataChecked []string
	writeAttempts   []string
	writtenMetadata map[string]summarizer.SummaryMetadata
}

func (s *fakeReportStore) ListCIDReportIDs(context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]string(nil), s.ids...), nil
}

func (s *fakeReportStore) LoadCIDReportFromStore(_ context.Context, cid string) (*summarizer.LoadedCIDReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loaded = append(s.loaded, cid)
	if err := s.loadErr[cid]; err != nil {
		return nil, err
	}
	report := s.reports[cid]
	if report == nil {
		report = loadedReport(cid, "hash-"+cid)
	}
	return report, nil
}

func (s *fakeReportStore) LoadSummaryMetadata(_ context.Context, cid string) (summarizer.SummaryMetadata, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadataChecked = append(s.metadataChecked, cid)
	if err := s.metadataErr[cid]; err != nil {
		return summarizer.SummaryMetadata{}, false, err
	}
	metadata, ok := s.metadata[cid]
	return metadata, ok, nil
}

func (s *fakeReportStore) WriteSummary(_ context.Context, cid string, _ string, metadata summarizer.SummaryMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeAttempts = append(s.writeAttempts, cid)
	if s.writtenMetadata == nil {
		s.writtenMetadata = make(map[string]summarizer.SummaryMetadata)
	}
	s.writtenMetadata[cid] = metadata
	return s.writeErr[cid]
}

type fakeNarrativeGenerator struct {
	mu         sync.Mutex
	errByCID   map[string]error
	summarized []string
}

func (g *fakeNarrativeGenerator) Summarize(_ context.Context, report *shared.CIDReport) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.summarized = append(g.summarized, report.CID)
	if err := g.errByCID[report.CID]; err != nil {
		return "", err
	}
	return "summary for " + report.CID, nil
}

type concurrencyTrackingGenerator struct {
	mu      sync.Mutex
	active  int
	maximum int
	delay   time.Duration
}

func (g *concurrencyTrackingGenerator) Summarize(ctx context.Context, report *shared.CIDReport) (string, error) {
	g.mu.Lock()
	g.active++
	if g.active > g.maximum {
		g.maximum = g.active
	}
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		g.active--
		g.mu.Unlock()
	}()
	timer := time.NewTimer(g.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "summary for " + report.CID, nil
	}
}

func (g *concurrencyTrackingGenerator) maximumConcurrency() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.maximum
}

type blockingNarrativeGenerator struct{}

func (blockingNarrativeGenerator) Summarize(ctx context.Context, _ *shared.CIDReport) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertStringSlices(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %v, want %v", got, want)
	}
}

func loadedReport(cid string, sourceSHA256 string) *summarizer.LoadedCIDReport {
	return &summarizer.LoadedCIDReport{
		Report:       &shared.CIDReport{CID: cid},
		SourceSHA256: sourceSHA256,
	}
}

func testGeneratedAt() time.Time {
	return time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
}
