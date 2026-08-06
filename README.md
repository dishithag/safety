# Summarizer

The summarizer is the core ZTA LLM Summaries implementation. It reads per-CID
audit JSON from an S3-compatible object store, deterministically analyzes the
report, optionally requests bounded guidance from Claude through GenAI Hub,
and writes one validated Markdown narrative per CID.

The batch entrypoint and worker orchestration live in `cmd/summarizer`.

## Package map

| File | Responsibility |
| --- | --- |
| `config.go` | Runtime configuration and defaults. |
| `report_loader.go` | JSON parsing and source SHA-256 calculation. |
| `report_store.go` | MinIO/S3 discovery, reads, writes, and freshness metadata. |
| `analysis.go` | Score validation and deterministic control selection. |
| `prompt.go` | Grounded structured-response and repair prompts. |
| `genaihub.go` | Anthropic SDK integration, platform chunking, and one repair attempt. |
| `guidance.go` | Structured response parsing, normalization, and validation. |
| `renderer.go` | Generated, placeholder, and fallback Markdown rendering. |
| `summary.go` | Object keys, summary version, profiles, and freshness matching. |
| `narrative_generator.go` | Provider selection. |

## Processing flow

1. List direct `reports/cids/{cid}.json` objects.
2. Load and hash each source report.
3. Skip a stored summary only when its source hash and generation profile match.
4. Validate report scores and select platform findings in Go.
5. Generate placeholder output or request Claude guidance in platform chunks.
6. Validate the structured guidance; attempt one repair after malformed output.
7. Render normal Markdown or a deterministic fallback.
8. Write `summary/cids/{cid}.md` with freshness metadata.

The batch processes multiple CIDs concurrently. A single CID failure does not
stop other reports, but a final fallback or failure tally makes the process exit
unsuccessfully so orchestration and monitoring can detect degradation.

## Providers

### Placeholder

`NARRATIVE_PROVIDER=placeholder` is the default. It exercises parsing,
selection, rendering, storage, API delivery, and the UI without using an LLM.

### GenAI Hub

`NARRATIVE_PROVIDER=genaihub` requires:

- `GENAI_HUB_MODEL`
- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_AUTH_TOKEN`

The Anthropic Go SDK calls the approved GenAI Hub-compatible endpoint. Claude
receives only prepared report facts and selected non-zero opportunities. Go
retains ownership of numeric facts, selection, ordering, and Markdown layout.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `S3_ENDPOINT` | `http://minio:9000` | S3-compatible endpoint. |
| `S3_BUCKET` | `dev` | Input and output bucket. |
| `S3_ACCESS_KEY` | `minioadmin` | Local access key. |
| `S3_SECRET_KEY` | `minioadmin` | Local secret key. |
| `NARRATIVE_PROVIDER` | `placeholder` | `placeholder` or `genaihub`. |
| `GENAI_HUB_MODEL` | none | Model identifier for GenAI Hub. |
| `GENAI_HUB_MAX_OUTPUT_TOKENS` | `16384` | Output limit for each Claude request. |
| `GENAI_HUB_PLATFORMS_PER_REQUEST` | `3` | Maximum platforms in one request. |
| `SUMMARIZER_CONCURRENCY` | `3` | Number of CIDs processed concurrently. |
| `SUMMARIZER_GENERATION_TIMEOUT` | `15m` | Total generation deadline for one CID. |

## Failure behavior

| Failure | Current action |
| --- | --- |
| Initial response is structurally invalid | Send one repair request. |
| Repaired response is invalid | Render deterministic fallback. |
| Repair invocation fails | Render deterministic fallback. |
| Network error, timeout, refusal, or truncation | Render deterministic fallback without a format-repair request. |
| Source report cannot be parsed or analyzed | Mark CID failed. |
| Object-store read or write fails | Mark CID failed. |
| Parent batch context is cancelled | Mark CID failed; do not create fallback. |

Fallback Markdown displays a guidance-status notice, preserves current source
facts, and is written with provider metadata `deterministic-fallback`. It is
therefore retried during a later `genaihub` run.

## Freshness metadata

Each Markdown object records:

- `source-sha256`
- `summary-version`
- `narrative-provider`
- `model` when applicable
- `generated-at`

Bump `SummaryVersion` in `summary.go` when a report-format change should
invalidate and regenerate existing objects.

## Tests

```shell
go test ./internal/summarizer ./cmd/summarizer
```

The suite covers report parsing, score validation, selection, prompt contracts,
structured guidance, repair behavior, chunking, rendering, freshness, fallback,
batch concurrency, timeouts, and failure isolation. The opt-in MinIO integration
test is documented in the repository [`README.md`](../../README.md).

