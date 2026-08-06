# summarizer (`cs.summarizer`)

One-shot batch executable that discovers per-CID ZTA audit reports, generates
or refreshes Markdown narratives, and writes them to MinIO/S3.

## Run

```shell
make run
go run ./cmd/summarizer
go build ./cmd/summarizer
```

The checked-in Kubernetes Job uses the offline `placeholder` provider. Live
guidance requires `NARRATIVE_PROVIDER=genaihub` plus the approved GenAI Hub
configuration described in [`internal/summarizer/README.md`](../../internal/summarizer/README.md).

## Batch result

The final structured log reports:

- `processed`: generated normally
- `fallback`: deterministic report written after GenAI guidance failed
- `skipped`: stored summary is still current
- `failed`: report could not be read, analyzed, or written

One CID does not stop the rest of the batch. The executable exits unsuccessfully
after the final tally if any CID failed or used fallback.

See the internal [summarizer README](../../internal/summarizer/README.md) for
configuration, freshness, retry/repair, fallback, and test details.

