# analyticsapi (cs.analyticsapi)

Customer-facing HTTP API that serves LLM-generated zerotrust audit narratives
to the front-end application.

The API reads Markdown narratives from the configured S3-compatible object
store. It never reads or parses source audit reports.

## Project

|                   |                                                          |
| ----------------- | -------------------------------------------------------- |
| Team Owner        | _TBD_                                                    |
| Primary Owner(s)  | _TBD_                                                    |
| Slack Channel     | _TBD_                                                    |
| Tickets           | _TBD_                                                    |
| Consumers         | Front-end application                                    |
| Related binary    | `cmd/summarizer` (produces the narratives this serves)   |

## Build & run

```shell
make run-api                 # run locally on :8080
go build ./cmd/analyticsapi  # build just this binary
```

## Configuration

All configuration is read from the environment (see
`internal/analyticsapi/domain/config.go`):

| Variable        | Default             | Purpose                            |
| --------------- | ------------------- | ---------------------------------- |
| `LISTEN_ADDR`   | `:8080`             | Host and port for the HTTP server. |
| `S3_ENDPOINT`   | `http://minio:9000` | Object store holding narratives.   |
| `S3_BUCKET`     | `dev`               | Bucket holding narratives.         |
| `S3_ACCESS_KEY` | `minioadmin`        | Local object-store access key.     |
| `S3_SECRET_KEY` | `minioadmin`        | Local object-store secret key.     |

## Endpoints

| Method & path                               | Description                                      |
| ------------------------------------------- | ------------------------------------------------ |
| `GET /healthz`                              | Liveness/readiness probe, returns `ok`.          |
| `GET /zero-trust-analytics/narratives/{id}` | Markdown narrative for the requested CID.        |

The narrative route returns `text/plain; charset=utf-8`. A missing narrative
returns `404`; other storage failures return `500`.

## Local setup

See the root [README](../../README.md) for the Tilt-based local workflow, which
builds this binary and deploys it to the local Colima Kubernetes cluster. With
the Tilt port-forward active:

```shell
curl -i http://localhost:8080/healthz
curl -i \
  http://localhost:8080/zero-trust-analytics/narratives/0000000000000000000000000000007b2
```
