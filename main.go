package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
	"go.crwd.dev/ce/zerotrust-analytics/internal/summarizer"
)

const serviceName = "cs.summarizer"

type reportStore interface {
	ListCIDReportIDs(context.Context) ([]string, error)
	SummaryExists(context.Context, string) (bool, error)
	LoadCIDReportFromStore(context.Context, string) (*shared.CIDReport, error)
	WriteSummary(context.Context, string, string) error
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

	stats, err := run(context.Background(), logger, store, generator, cfg.NarrativeProvider)
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

func run(ctx context.Context, logger *slog.Logger, store reportStore, generator narrativeGenerator, provider string) (runStats, error) {
	ids, err := store.ListCIDReportIDs(ctx)
	if err != nil {
		return runStats{}, fmt.Errorf("list cid reports: %w", err)
	}

	stats := runStats{Total: len(ids)}
	if len(ids) == 0 {
		logger.Info("no cid reports found")
		return stats, nil
	}

	provider = effectiveNarrativeProvider(provider)
	for _, cid := range ids {
		exists, err := store.SummaryExists(ctx, cid)
		if err != nil {
			stats.Failed++
			logger.Error("failed to check existing summary", "cid", cid, "err", err)
			continue
		}
		if exists {
			stats.Skipped++
			logger.Info("skipping already summarized cid report", "cid", cid, "summary_key", summaryKey(cid))
			continue
		}

		report, err := store.LoadCIDReportFromStore(ctx, cid)
		if err != nil {
			stats.Failed++
			logger.Error("failed to load cid report", "cid", cid, "err", err)
			continue
		}

		summary, err := generator.Summarize(ctx, report)
		if err != nil {
			stats.Failed++
			logger.Error("failed to summarize cid report", "cid", cid, "err", err)
			continue
		}
		if err := store.WriteSummary(ctx, report.CID, summary); err != nil {
			stats.Failed++
			logger.Error("failed to write summary", "cid", report.CID, "err", err)
			continue
		}

		stats.Processed++
		logger.Info(
			"loaded cid report and wrote summary to object store",
			"available_reports", len(ids),
			"cid", report.CID,
			"platforms", len(report.Platforms),
			"narrative_provider", provider,
			"summary_key", summaryKey(report.CID),
		)
	}

	return stats, nil
}

func effectiveNarrativeProvider(provider string) string {
	if provider == "" {
		return "placeholder"
	}
	return provider
}

func summaryKey(cid string) string {
	return "summary/cids/" + cid + ".md"
}
