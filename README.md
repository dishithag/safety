# zerotrust-analytics

A Go service repository following the same layered (clean / hexagonal) architecture
as the sibling `zerotrust` repo. This document explains **what goes where and why** so
new contributors can place code consistently.

## Module

```
module go.crwd.dev/ce/zerotrust-analytics
```

Import internal packages via their full module path, e.g.
`go.crwd.dev/ce/zerotrust-analytics/domain`.

## Repository layout

```
.
├── cmd/                  Executable entrypoints — one subdir per binary
│   ├── summarizer/       LLM audit-report summarizer (main.go + main_test.go)
│   └── analyticsapi/     HTTP API serving narratives to the front-end
├── demo-ui/              Isolated React/Vite report and analytics demo
├── deploy/               Local Kubernetes manifests and MinIO seed image
├── internal/             Private application code (cannot be imported externally)
│   ├── common/           Shared internal infrastructure (datastore clients, etc.)
│   └── <service>/        Per-service code, split into the layers below
│       ├── domain/       Service-specific entities, interfaces, config
│       ├── usecase/      Application/business logic (orchestrates domain + gateways)
│       ├── gateways/     Adapters to the outside world (HTTP API, message bus)
│       └── validation/   Request validation rules
├── domain/               Shared domain models, mappers, and business rules
├── common/               Shared, dependency-light utilities (types, helpers)
├── gateways/             Shared adapters to external systems
│   └── repos/            Repository implementations (data store access)
├── scripts/              One-off and code-generation programs
├── tools/                Build/dev tool dependencies (pinned via tools.go)
├── testdata/             Sample fixtures for tests and local runs (see below)
└── docs/                 Architecture diagrams and design docs
```

## Layering rules (why it's organized this way)

The architecture isolates business logic from delivery and infrastructure so each
layer can change and be tested independently. Dependencies point **inward**:

```
cmd  ──▶  internal/<service>  ──▶  domain
                │                    ▲
                ├── usecase ─────────┘   (pure logic, no I/O)
                ├── gateways ──▶ external systems (DB, HTTP, Kafka)
                └── validation
```

- **`cmd/`** — thin `main` packages. Wire dependencies and start a service; keep
  logic out of here so it stays testable. Each subdir builds one binary.
- **`internal/`** — everything private to this repo. Go's `internal` rule prevents
  other modules importing it, keeping the public surface intentional.
- **`domain/` (and `internal/<service>/domain`)** — entities, value objects, and
  the interfaces gateways must satisfy. No framework or I/O imports here.
- **`usecase/`** — orchestrates domain objects to fulfil a use case. Depends on
  domain interfaces, never on concrete gateways.
- **`gateways/` / `gateways/repos/`** — implement the domain interfaces against
  real systems (databases, APIs). This is where I/O lives.
- **`common/`** — small reusable helpers shared across services. Keep dependencies
  minimal so anything can import it without cycles.
- **`scripts/` & `tools/`** — supporting programs and pinned tooling, kept out of
  the production build graph.

## Adding a new service

1. Create `cmd/<service>/main.go` for the entrypoint.
2. Create `internal/<service>/` with `domain/`, `usecase/`, `gateways/`, and
   (if it accepts external input) `validation/`.
3. Put cross-service models in top-level `domain/` and shared helpers in `common/`.
4. Add a `README.md` in `internal/<service>/` describing the service (owners,
   infrastructure, dependencies) — see the `zerotrust` repo for the template.

## Common commands

```shell
make build   # compile all packages
make test    # run unit tests
make vet     # run go vet
make run     # run the summarizer binary
make run-api # run the analyticsapi HTTP server (:8080)
```

Frontend checks and local development:

```shell
cd demo-ui
npm ci
npm run check:catalog
npm run check:trends
npm run build
npm run dev
```

See [`docs/PROJECT_HANDOFF.md`](docs/PROJECT_HANDOFF.md) for a concise system
handoff and [`docs/README.md`](docs/README.md) for the complete documentation
index.

## Local development with Tilt

[Tilt](https://tilt.dev) builds the service images (`summarizer`, `analyticsapi`)
and deploys them, plus a minio S3-compatible object store, onto the
local Colima Kubernetes cluster. Everything lives in a dedicated
`zerotrust-analytics` namespace so it never collides with other workloads on the
cluster.

### Prerequisites

You need the following installed locally. On macOS everything is available via
[Homebrew](https://brew.sh):

```shell
brew install colima docker kubectl tilt-dev/tap/tilt
```

| Tool      | Purpose                                                        |
| --------- | -------------------------------------------------------------- |
| `colima`  | Runs the local VM hosting Docker + a k3s Kubernetes cluster.   |
| `docker`  | Docker CLI used by Tilt to build the service images.           |
| `kubectl` | Talks to the cluster; Tilt also uses your current kube context.|
| `tilt`    | Orchestrates the build + deploy loop (v0.36+ recommended).     |

> Helm is **not** required — minio is deployed from plain manifests in `deploy/`.

### One-time configuration

1. **Start Colima with Kubernetes and the Docker runtime.** The Docker runtime
   is important: it lets the cluster use images Tilt builds locally, so no
   registry push is needed.

   ```shell
   colima start --kubernetes --runtime docker
   ```

   Give it more resources if the defaults are tight, e.g.
   `colima start --kubernetes --runtime docker --cpu 4 --memory 8`.

2. **Point kubectl at the Colima cluster.** Colima sets this up automatically;
   confirm your current context is `colima`:

   ```shell
   kubectl config current-context   # should print: colima
   kubectl get nodes                # node should be Ready
   ```

   If it isn't, run `kubectl config use-context colima`.

### Running

From the repo root:

```shell
tilt up        # build + deploy, with a live web UI at http://localhost:10350
tilt down      # tear everything down
```

To verify a one-shot build + deploy without the interactive UI (useful in CI):

```shell
tilt ci        # builds, deploys, waits for everything healthy, then exits
```

What gets deployed (see `Tiltfile` and `deploy/`):

- **summarizer** — the `cmd/summarizer` binary, built via `Dockerfile` and run
  as a one-shot Job after `minio-setup` finishes. It discovers every per-CID
  report, generates a Markdown narrative, and writes it to minio. The checked-in
  deployment uses `NARRATIVE_PROVIDER=placeholder` so local infrastructure and
  batch behavior can be tested without spending LLM tokens. Set the provider to
  `genaihub` and supply the approved GenAI Hub configuration for live narratives.
- **analyticsapi** — the `cmd/analyticsapi` HTTP server, a long-running
  Deployment that reads generated Markdown from minio and serves it to the
  front-end. Tilt port-forwards http://localhost:8080.
  - `GET /healthz`
  - `GET /zero-trust-analytics/narratives/{id}`
- **minio** — an S3-compatible object store. Tilt port-forwards:
  - S3 API → http://localhost:9000
  - web console → http://localhost:9001 (login `minioadmin` / `minioadmin`)
  - a `minio-setup` Job creates the `dev` bucket and seeds it with the sample
    audit reports (see [Test data](#test-data) below).

The React/Vite demo UI is intentionally not deployed by Tilt. After the Tilt
resources are ready, start it separately with `cd demo-ui && npm run dev`, then
open http://localhost:5173.

When `NARRATIVE_PROVIDER=genaihub`, request sizing is controlled independently
from CID batch concurrency:

| Variable | Default | Purpose |
| --- | ---: | --- |
| `GENAI_HUB_MAX_OUTPUT_TOKENS` | `16384` | Maximum output tokens for each Claude request. |
| `GENAI_HUB_PLATFORMS_PER_REQUEST` | `3` | Maximum report platforms included in one Claude request. |
| `SUMMARIZER_GENERATION_TIMEOUT` | `15m` | Maximum total generation time for one CID across all requests. |

Larger reports are analyzed once and divided into platform groups. Every Claude
response is normalized and validated against its supplied facts. A structurally
invalid response receives one bounded repair request. Go then combines all
validated platform guidance in source order and renders one
`summary/cids/{cid}.md` object. If generated guidance remains unavailable, the
summarizer writes a deterministic fallback containing current scores and control
selections instead of leaving the CID without Markdown.

Zero-compliance controls are always listed by Go with a stable set of common
diagnostic possibilities. They are not sent to Claude for grouping or
remediation because aggregate `0%` coverage does not reveal the underlying
cause. Claude guidance is limited to the selected non-zero improvement
opportunities.

Colima's k3s uses the docker runtime, so images built by Tilt are visible to
the cluster directly — no registry push is required.

### Verify the full loop

Wait for `minio`, `minio-setup`, `summarizer`, and `analyticsapi` to become
healthy in Tilt. The summarizer logs a final
`processed / fallback / skipped / failed` tally. Then verify the API from
another terminal:

```shell
curl -i http://localhost:8080/healthz

curl -i \
  http://localhost:8080/zero-trust-analytics/narratives/0000000000000000000000000000007b2
```

The narrative request returns `200 OK` with a generated Markdown body and a
`Content-Type` of `text/plain; charset=utf-8`. An unknown CID returns `404`:

```shell
curl -i \
  http://localhost:8080/zero-trust-analytics/narratives/does-not-exist
```

Summaries are current for a source-report SHA-256, summary format version,
narrative provider, and model. A rerun skips matching summaries, regenerates a
summary when any of those inputs change, continues past individual report
failures, and exits unsuccessfully after publishing the final degraded/failure
tally if any report failed or required fallback. Fallback objects use the
`deterministic-fallback` provider metadata, so they never satisfy the requested
`genaihub` profile and are retried on a later run.

### Test against local minio

The regular unit suite does not require external services:

```shell
make test
```

With Tilt's minio port-forward running on `localhost:9000`, run the opt-in
integration test:

```shell
RUN_MINIO_INTEGRATION=1 \
S3_ENDPOINT=http://localhost:9000 \
go test ./cmd/summarizer -run '^TestMinIOPipelineEndToEnd$' -v
```

The test creates and removes a unique temporary bucket. It seeds one report,
runs the batch with a fake LLM, verifies Markdown and freshness metadata,
retrieves the narrative through the HTTP handler, and confirms a second run is
skipped.

## Test data

`testdata/sample_audit_reports/` holds fake audit reports for unit tests and
manual runs. The layout mirrors the production S3 key structure, minus the
`reports/` prefix (the directory itself stands in for it):

```
testdata/sample_audit_reports/
├── cloud_audit.json          → reports/cloud_audit.json   (cloud rollup)
├── cids/{cid}.json           → reports/cids/{cid}.json     (per-CID audit report)
└── generate.py               generator script (not uploaded)
```

Production also writes weekly text reports at `reports/{year}_week{NN}.txt`
(ISO year + zero-padded ISO week); the sample set does not include these.

On `tilt up`, the `minio-setup` Job mirrors these files into the `dev` bucket
under `reports/`, so the in-cluster object store matches production keys:

| Artifact            | S3 key (in `dev` bucket)        |
| ------------------- | ------------------------------- |
| Per-CID audit report| `reports/cids/{cid}.json`       |
| Cloud rollup        | `reports/cloud_audit.json`      |
| Generated narrative | `summary/cids/{cid}.md`          |

Browse them in the minio console (http://localhost:9001) or with `mc` against
the port-forwarded API at http://localhost:9000. Freshness information is stored
as object metadata on each Markdown narrative rather than as separate sidecar
objects.
