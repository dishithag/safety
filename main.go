package main

import (
	"context"
	"log/slog"
	"os"

	"go.crwd.dev/ce/zerotrust-analytics/internal/summarizer"
)

const serviceName = "cs.summarizer"

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

	ctx := context.Background()
	ids, err := store.ListCIDReportIDs(ctx)
	if err != nil {
		logger.Error("failed to list cid reports", "err", err)
		os.Exit(1)
	}
	if len(ids) == 0 {
		logger.Info("no cid reports found")
		return
	}

	report, err := store.LoadCIDReportFromStore(ctx, ids[0])
	if err != nil {
		logger.Error("failed to load cid report", "cid", ids[0], "err", err)
		os.Exit(1)
	}

	summary, err := generator.Summarize(ctx, report)
	if err != nil {
		logger.Error("failed to summarize cid report", "cid", report.CID, "err", err)
		os.Exit(1)
	}
	if err := store.WriteSummary(ctx, report.CID, summary); err != nil {
		logger.Error("failed to write summary", "cid", report.CID, "err", err)
		os.Exit(1)
	}

	logger.Info(
		"loaded cid report and wrote summary to object store",
		"available_reports", len(ids),
		"cid", report.CID,
		"platforms", len(report.Platforms),
		"narrative_provider", cfg.NarrativeProvider,
		"summary_key", "summary/cids/"+report.CID+".md",
	)
}
