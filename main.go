package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
	"go.crwd.dev/ce/zerotrust-analytics/internal/summarizer"
)

const serviceName = "cs.summarizer"

type reportStore interface {
	ListCIDReportIDs(context.Context) ([]string, error)
	LoadCIDReportFromStore(context.Context, string) (*summarizer.LoadedCIDReport, error)
	LoadSummaryMetadata(context.Context, string) (summarizer.SummaryMetadata, bool, error)
	WriteSummary(context.Context, string, string, summarizer.SummaryMetadata) error
}

type narrativeGenerator interface {
	Summarize(context.Context, *shared.CIDReport) (string, error)
}

type runStats struct {
	Total     int
	Processed int
	Skipped   int
	Failed    int
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", serviceName)
	cfg := summarizer.LoadConfig(serviceName)

	store, err := summarizer.NewReportStore(cfg)
	if err != nil {
		logger.Error("failed to initialize report store", "err", err)
		os.Exit(1)
	}
	generator, err := summarizer.NewNarrativeGenerator(cfg)
	if err != nil {
		logger.Error("failed to initialize narrative generator", "err", err)
		os.Exit(1)
	}

	profile := summarizer.CurrentSummaryProfile(cfg)
	stats, err := run(context.Background(), logger, store, generator, profile)
	if err != nil {
		logger.Error("summarizer run failed", "err", err)
		os.Exit(1)
	}

	logger.Info(
		"summarizer run complete",
		"total", stats.Total,
		"processed", stats.Processed,
		"skipped", stats.Skipped,
		"failed", stats.Failed,
	)
}

func run(ctx context.Context, logger *slog.Logger, store reportStore, generator narrativeGenerator, profile summarizer.SummaryProfile) (runStats, error) {
	ids, err := store.ListCIDReportIDs(ctx)
	if err != nil {
		return runStats{}, fmt.Errorf("list cid reports: %w", err)
	}

	stats := runStats{Total: len(ids)}
	if len(ids) == 0 {
		logger.Info("no cid reports found")
		return stats, nil
	}

	for _, cid := range ids {
		loaded, err := store.LoadCIDReportFromStore(ctx, cid)
		if err != nil {
			stats.Failed++
			logger.Error("failed to load cid report", "cid", cid, "err", err)
			continue
		}

		metadata, exists, err := store.LoadSummaryMetadata(ctx, cid)
		if err != nil {
			stats.Failed++
			logger.Error("failed to load summary metadata", "cid", cid, "err", err)
			continue
		}
		if exists && metadata.Matches(loaded.SourceSHA256, profile) {
			stats.Skipped++
			logger.Info(
				"skipping current cid report summary",
				"cid", cid,
				"summary_key", summarizer.SummaryObjectKey(cid),
				"metadata_key", summarizer.SummaryMetadataObjectKey(cid),
				"source_sha256", loaded.SourceSHA256,
				"summary_version", profile.Version,
				"narrative_provider", profile.NarrativeProvider,
				"model", profile.Model,
			)
			continue
		}

		summary, err := generator.Summarize(ctx, loaded.Report)
		if err != nil {
			stats.Failed++
			logger.Error("failed to summarize cid report", "cid", cid, "err", err)
			continue
		}
		metadata = summarizer.NewSummaryMetadata(loaded.SourceSHA256, profile, time.Now())
		if err := store.WriteSummary(ctx, cid, summary, metadata); err != nil {
			stats.Failed++
			logger.Error("failed to write summary", "cid", cid, "err", err)
			continue
		}

		stats.Processed++
		logger.Info(
			"loaded cid report and wrote summary to object store",
			"available_reports", len(ids),
			"cid", cid,
			"report_cid", loaded.Report.CID,
			"platforms", len(loaded.Report.Platforms),
			"source_sha256", loaded.SourceSHA256,
			"summary_version", profile.Version,
			"narrative_provider", profile.NarrativeProvider,
			"model", profile.Model,
			"summary_key", summarizer.SummaryObjectKey(cid),
			"metadata_key", summarizer.SummaryMetadataObjectKey(cid),
		)
	}

	return stats, nil
}
