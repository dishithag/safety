package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
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
	Fallback  int
	Skipped   int
	Failed    int
}

type runOptions struct {
	Concurrency       int
	GenerationTimeout time.Duration
}

type reportStatus uint8

const (
	reportProcessed reportStatus = iota
	reportFallback
	reportSkipped
	reportFailed
)

type reportResult struct {
	status reportStatus
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", serviceName)
	cfg, err := summarizer.LoadConfig(serviceName)
	if err != nil {
		logger.Error("failed to load summarizer configuration", "err", err)
		os.Exit(1)
	}

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
	started := time.Now()
	stats, err := runWithOptions(
		context.Background(),
		logger,
		store,
		generator,
		profile,
		runOptions{
			Concurrency:       cfg.Concurrency,
			GenerationTimeout: cfg.GenerationTimeout,
		},
	)
	if err != nil {
		logger.Error("summarizer run failed", "err", err)
		os.Exit(1)
	}

	logger.Info(
		"summarizer run complete",
		"total", stats.Total,
		"processed", stats.Processed,
		"fallback", stats.Fallback,
		"skipped", stats.Skipped,
		"failed", stats.Failed,
		"duration_ms", time.Since(started).Milliseconds(),
	)
	if err := batchCompletionError(stats); err != nil {
		logger.Error("summarizer run completed with degraded reports or failures", "err", err)
		os.Exit(1)
	}
}

func batchCompletionError(stats runStats) error {
	if stats.Failed == 0 && stats.Fallback == 0 {
		return nil
	}
	if stats.Failed == 0 {
		return fmt.Errorf("%d of %d cid reports used deterministic fallback", stats.Fallback, stats.Total)
	}
	if stats.Fallback == 0 {
		return fmt.Errorf("%d of %d cid reports failed", stats.Failed, stats.Total)
	}
	return fmt.Errorf(
		"%d of %d cid reports failed and %d used deterministic fallback",
		stats.Failed,
		stats.Total,
		stats.Fallback,
	)
}

func run(ctx context.Context, logger *slog.Logger, store reportStore, generator narrativeGenerator, profile summarizer.SummaryProfile) (runStats, error) {
	return runWithOptions(ctx, logger, store, generator, profile, runOptions{Concurrency: 1})
}

func runWithOptions(
	ctx context.Context,
	logger *slog.Logger,
	store reportStore,
	generator narrativeGenerator,
	profile summarizer.SummaryProfile,
	options runOptions,
) (runStats, error) {
	ids, err := store.ListCIDReportIDs(ctx)
	if err != nil {
		return runStats{}, fmt.Errorf("list cid reports: %w", err)
	}

	stats := runStats{Total: len(ids)}
	if len(ids) == 0 {
		logger.Info("no cid reports found")
		return stats, nil
	}

	workerCount := options.Concurrency
	if workerCount <= 0 {
		workerCount = 1
	}
	if workerCount > len(ids) {
		workerCount = len(ids)
	}
	logger.Info(
		"starting summarizer batch",
		"total", len(ids),
		"concurrency", workerCount,
		"generation_timeout", options.GenerationTimeout.String(),
	)

	jobs := make(chan string)
	results := make(chan reportResult)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for cid := range jobs {
				results <- processCID(ctx, logger, store, generator, profile, options, len(ids), cid)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, cid := range ids {
			select {
			case jobs <- cid:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	for result := range results {
		switch result.status {
		case reportProcessed:
			stats.Processed++
		case reportFallback:
			stats.Fallback++
		case reportSkipped:
			stats.Skipped++
		case reportFailed:
			stats.Failed++
		}
		logger.Info(
			"summarizer batch progress",
			"completed", stats.Processed+stats.Fallback+stats.Skipped+stats.Failed,
			"total", stats.Total,
			"processed", stats.Processed,
			"fallback", stats.Fallback,
			"skipped", stats.Skipped,
			"failed", stats.Failed,
		)
	}
	if err := ctx.Err(); err != nil {
		return stats, fmt.Errorf("summarizer batch canceled: %w", err)
	}

	return stats, nil
}

func processCID(
	ctx context.Context,
	logger *slog.Logger,
	store reportStore,
	generator narrativeGenerator,
	profile summarizer.SummaryProfile,
	options runOptions,
	total int,
	cid string,
) reportResult {
	started := time.Now()
	phaseStarted := time.Now()
	loaded, err := store.LoadCIDReportFromStore(ctx, cid)
	loadDuration := time.Since(phaseStarted)
	if err != nil {
		logger.Error(
			"failed to load cid report",
			"cid", cid,
			"phase", "load",
			"load_ms", loadDuration.Milliseconds(),
			"total_ms", time.Since(started).Milliseconds(),
			"err", err,
		)
		return reportResult{status: reportFailed}
	}

	phaseStarted = time.Now()
	metadata, exists, err := store.LoadSummaryMetadata(ctx, cid)
	metadataDuration := time.Since(phaseStarted)
	if err != nil {
		logger.Error(
			"failed to load summary metadata",
			"cid", cid,
			"phase", "metadata",
			"load_ms", loadDuration.Milliseconds(),
			"metadata_ms", metadataDuration.Milliseconds(),
			"total_ms", time.Since(started).Milliseconds(),
			"err", err,
		)
		return reportResult{status: reportFailed}
	}
	if exists && metadata.Matches(loaded.SourceSHA256, profile) {
		logger.Info(
			"skipping current cid report summary",
			"cid", cid,
			"summary_key", summarizer.SummaryObjectKey(cid),
			"source_sha256", loaded.SourceSHA256,
			"summary_version", profile.Version,
			"narrative_provider", profile.NarrativeProvider,
			"model", profile.Model,
			"load_ms", loadDuration.Milliseconds(),
			"metadata_ms", metadataDuration.Milliseconds(),
			"total_ms", time.Since(started).Milliseconds(),
		)
		return reportResult{status: reportSkipped}
	}

	generationCtx := ctx
	cancelGeneration := func() {}
	if options.GenerationTimeout > 0 {
		generationCtx, cancelGeneration = context.WithTimeout(ctx, options.GenerationTimeout)
	}
	phaseStarted = time.Now()
	summary, err := generator.Summarize(generationCtx, loaded.Report)
	generationDuration := time.Since(phaseStarted)
	generationContextErr := generationCtx.Err()
	cancelGeneration()
	generationErr := err
	fallbackUsed := false
	if err != nil {
		if profile.NarrativeProvider == "genaihub" && ctx.Err() == nil {
			analysis, fallbackErr := summarizer.AnalyzeReport(loaded.Report)
			if fallbackErr == nil {
				summary, fallbackErr = summarizer.RenderFallbackSummary(analysis)
			}
			if fallbackErr == nil {
				fallbackUsed = true
				err = nil
				logger.Warn(
					"generated guidance unavailable; using deterministic fallback summary",
					"cid", cid,
					"phase", "generation",
					"timed_out", errors.Is(generationContextErr, context.DeadlineExceeded),
					"generation_ms", generationDuration.Milliseconds(),
					"generation_error", generationErr,
				)
			} else {
				err = fmt.Errorf("generated guidance failed: %w; render deterministic fallback: %v", generationErr, fallbackErr)
			}
		}
		if err != nil {
			logger.Error(
				"failed to summarize cid report",
				"cid", cid,
				"phase", "generation",
				"timed_out", errors.Is(generationContextErr, context.DeadlineExceeded),
				"load_ms", loadDuration.Milliseconds(),
				"metadata_ms", metadataDuration.Milliseconds(),
				"generation_ms", generationDuration.Milliseconds(),
				"total_ms", time.Since(started).Milliseconds(),
				"err", err,
			)
			return reportResult{status: reportFailed}
		}
	}

	writeProfile := profile
	if fallbackUsed {
		writeProfile = summarizer.FallbackSummaryProfile(profile)
	}
	metadata = summarizer.NewSummaryMetadata(loaded.SourceSHA256, writeProfile, time.Now())
	phaseStarted = time.Now()
	err = store.WriteSummary(ctx, cid, summary, metadata)
	writeDuration := time.Since(phaseStarted)
	if err != nil {
		logger.Error(
			"failed to write summary",
			"cid", cid,
			"phase", "write",
			"load_ms", loadDuration.Milliseconds(),
			"metadata_ms", metadataDuration.Milliseconds(),
			"generation_ms", generationDuration.Milliseconds(),
			"fallback", fallbackUsed,
			"write_ms", writeDuration.Milliseconds(),
			"total_ms", time.Since(started).Milliseconds(),
			"err", err,
		)
		return reportResult{status: reportFailed}
	}

	if fallbackUsed {
		logger.Warn(
			"wrote deterministic fallback summary to object store",
			"available_reports", total,
			"cid", cid,
			"report_cid", loaded.Report.CID,
			"platforms", len(loaded.Report.Platforms),
			"source_sha256", loaded.SourceSHA256,
			"summary_version", writeProfile.Version,
			"narrative_provider", writeProfile.NarrativeProvider,
			"summary_key", summarizer.SummaryObjectKey(cid),
			"load_ms", loadDuration.Milliseconds(),
			"metadata_ms", metadataDuration.Milliseconds(),
			"generation_ms", generationDuration.Milliseconds(),
			"write_ms", writeDuration.Milliseconds(),
			"total_ms", time.Since(started).Milliseconds(),
			"generation_error", generationErr,
		)
		return reportResult{status: reportFallback}
	}

	logger.Info(
		"loaded cid report and wrote summary to object store",
		"available_reports", total,
		"cid", cid,
		"report_cid", loaded.Report.CID,
		"platforms", len(loaded.Report.Platforms),
		"source_sha256", loaded.SourceSHA256,
		"summary_version", profile.Version,
		"narrative_provider", profile.NarrativeProvider,
		"model", profile.Model,
		"summary_key", summarizer.SummaryObjectKey(cid),
		"load_ms", loadDuration.Milliseconds(),
		"metadata_ms", metadataDuration.Milliseconds(),
		"generation_ms", generationDuration.Milliseconds(),
		"write_ms", writeDuration.Milliseconds(),
		"total_ms", time.Since(started).Milliseconds(),
	)
	return reportResult{status: reportProcessed}
}
