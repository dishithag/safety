# cs.analyticsapi

## Overview

Purpose: customer-facing HTTP API that serves LLM-generated zerotrust audit
narratives (produced by the `summarizer` binary and stored in S3) to the
front-end application.

The implementation is S3-backed and returns the stored Markdown without reading
or parsing source audit reports.

|                     |                                                       |
| :------------------ | :---------------------------------------------------- |
| Team Owner:         | _TBD_                                                 |
| Primary Owner(s):   | _TBD_                                                 |
| Questions:          | _TBD_                                                 |
| Supported Clouds:   | _TBD_                                                 |

## Architecture

Layered (clean/hexagonal), dependencies pointing inward:

```
cmd/analyticsapi/main.go        thin entrypoint: load config, wire, serve
└── internal/analyticsapi/
    ├── service.go              owns the HTTP server lifecycle
    ├── domain/config.go        env-sourced configuration (no I/O)
    ├── domain/narrative.go     storage interface and not-found error
    ├── usecase/narrative/      application logic: fetch a narrative
    └── gateways/
        ├── api/                HTTP delivery: routes -> usecase
        └── repos/              MinIO/S3 narrative-store implementation
```

The `gateways/api` layer depends on the `NarrativeService` interface, which the
`usecase/narrative` package satisfies — never the reverse.

## Endpoints

| Method & path                               | Description                               |
| ------------------------------------------- | ----------------------------------------- |
| `GET /healthz`                              | Liveness/readiness probe, returns `ok`.   |
| `GET /zero-trust-analytics/narratives/{id}` | Stored Markdown for the requested CID.    |

## Configuration

Environment-sourced (see `domain/config.go`): `LISTEN_ADDR` (`:8080`),
`S3_ENDPOINT` (`http://minio:9000`), `S3_BUCKET` (`dev`), `S3_ACCESS_KEY`
(`minioadmin`), and `S3_SECRET_KEY` (`minioadmin`).

## Consumers

Front-end application.

## Dependencies

Reads `summary/cids/{cid}.md` from S3 (minio in local development). No other
service dependencies.

## Testing

Unit tests cover HTTP status mapping and the S3 repository's missing-object
behavior. The opt-in end-to-end test in `cmd/summarizer` exercises the real
repository and HTTP handler against local minio. Run commands are documented in
the root [README](../../README.md).

## Local setup

Covered by the root [README](../../README.md) Tilt workflow — `tilt up` builds
this binary and deploys it to the local Colima Kubernetes cluster, port-forwarded
to `localhost:8080`.
