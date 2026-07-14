package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestServiceName(t *testing.T) {
	if serviceName == "" {
		t.Fatal("serviceName should not be empty")
	}
}

func TestRunProcessesAllReportsAndContinuesAfterPerCIDFailures(t *testing.T) {
	store := &fakeReportStore{
		ids: []string{"new", "existing", "bad-load", "bad-summary", "bad-write"},
		existing: map[string]bool{
			"existing": true,
		},
		reports: map[string]*shared.CIDReport{
			"new":         {CID: "new"},
			"bad-summary": {CID: "bad-summary"},
			"bad-write":   {CID: "bad-write"},
		},
		loadErr: map[string]error{
			"bad-load": errors.New("load failed"),
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

	stats, err := run(context.Background(), discardLogger(), store, generator, "placeholder")
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	want := runStats{Total: 5, Processed: 1, Skipped: 1, Failed: 3}
	if stats != want {
		t.Fatalf("stats = %+v, want %+v", stats, want)
	}

	assertStringSlices(t, store.checked, []string{"new", "existing", "bad-load", "bad-summary", "bad-write"})
	assertStringSlices(t, store.loaded, []string{"new", "bad-load", "bad-summary", "bad-write"})
	assertStringSlices(t, generator.summarized, []string{"new", "bad-summary", "bad-write"})
	assertStringSlices(t, store.writeAttempts, []string{"new", "bad-write"})
}

func TestRunReturnsListError(t *testing.T) {
	store := &fakeReportStore{listErr: errors.New("list failed")}

	stats, err := run(context.Background(), discardLogger(), store, &fakeNarrativeGenerator{}, "placeholder")
	if err == nil {
		t.Fatal("expected list error")
	}
	if stats != (runStats{}) {
		t.Fatalf("stats = %+v, want zero value", stats)
	}
}

func TestRunNoReports(t *testing.T) {
	stats, err := run(context.Background(), discardLogger(), &fakeReportStore{}, &fakeNarrativeGenerator{}, "")
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}
	if stats != (runStats{}) {
		t.Fatalf("stats = %+v, want zero value", stats)
	}
}

type fakeReportStore struct {
	ids           []string
	listErr       error
	existing      map[string]bool
	existsErr     map[string]error
	reports       map[string]*shared.CIDReport
	loadErr       map[string]error
	writeErr      map[string]error
	checked       []string
	loaded        []string
	writeAttempts []string
}

func (s *fakeReportStore) ListCIDReportIDs(context.Context) ([]string, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.ids, nil
}

func (s *fakeReportStore) SummaryExists(_ context.Context, cid string) (bool, error) {
	s.checked = append(s.checked, cid)
	if err := s.existsErr[cid]; err != nil {
		return false, err
	}
	return s.existing[cid], nil
}

func (s *fakeReportStore) LoadCIDReportFromStore(_ context.Context, cid string) (*shared.CIDReport, error) {
	s.loaded = append(s.loaded, cid)
	if err := s.loadErr[cid]; err != nil {
		return nil, err
	}
	report := s.reports[cid]
	if report == nil {
		report = &shared.CIDReport{CID: cid}
	}
	return report, nil
}

func (s *fakeReportStore) WriteSummary(_ context.Context, cid string, _ string) error {
	s.writeAttempts = append(s.writeAttempts, cid)
	return s.writeErr[cid]
}

type fakeNarrativeGenerator struct {
	errByCID   map[string]error
	summarized []string
}

func (g *fakeNarrativeGenerator) Summarize(_ context.Context, report *shared.CIDReport) (string, error) {
	g.summarized = append(g.summarized, report.CID)
	if err := g.errByCID[report.CID]; err != nil {
		return "", err
	}
	return "summary for " + report.CID, nil
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
