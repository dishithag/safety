# Zero Trust Analytics Demo UI

Standalone frontend for browsing generated per-CID Markdown narratives. This
folder is intentionally isolated from the Go services and is not part of the
production deployment.

## Run locally

Start MinIO and the existing analytics API on `http://localhost:8080`, then run:

```shell
cd demo-ui
npm ci
npm run dev
```

Open `http://localhost:5173`.

The local defaults match the repository's MinIO setup:

```text
DEMO_MINIO_ENDPOINT=http://localhost:9000
DEMO_MINIO_BUCKET=dev
DEMO_MINIO_ACCESS_KEY=minioadmin
DEMO_MINIO_SECRET_KEY=minioadmin
```

Override any value in `demo-ui/.env.local` when the work-laptop setup differs.
The demo server also accepts the existing `S3_ENDPOINT`, `S3_BUCKET`,
`S3_ACCESS_KEY`, and `S3_SECRET_KEY` names. MinIO credentials stay in the Vite
server and are never sent to the browser.

Vite proxies narrative requests to the existing Analytics API, so no backend
CORS change is required. To use another API origin, add:

```text
VITE_API_BASE_URL=http://localhost:8080
```

Direct cross-origin requests require that API origin to permit browser CORS.

## Report data flow

The Reports sidebar is generated from the Markdown objects currently stored
under `summary/cids/` in MinIO:

1. The demo-only Vite route `GET /demo-api/narratives` lists direct
   `summary/cids/{cid}.md` objects in MinIO.
2. The browser uses that list to build the sidebar.
3. Selecting a CID fetches its Markdown through the existing
   `GET /zero-trust-analytics/narratives/{cid}` Analytics API endpoint.

The list refreshes every 15 seconds, when the browser regains focus, and when
the Reports tab is reopened. A newly generated Markdown object appears
automatically, a deleted object disappears, and an overwritten object causes
the selected narrative to reload. Sidecar JSON, nested objects, and unrelated
keys are ignored.

Entering a CID directly still calls the existing narrative endpoint. Stored
Markdown is rendered, while a missing object is shown as unavailable after the
API returns `404`.

This catalog route exists entirely in `demo-ui/`; no Go API, Tilt, or deployment
configuration is changed.

## Analytics scope

The separate Analytics tab uses checked-in synthetic eight-CID history for a
stable demo. It includes:

- A seven-day synthetic overall-insights view.
- Median and device-weighted posture, current score distribution, dedicated
  zero-compliance and recurring-gap rankings, OS/sensor balance, platform
  posture, and CID movement.
- Per-CID score and platform trends, device-volume context, zero-control
  transitions, and exact control movement.
- A preview of the grounded weekly narrative expected from a future scheduled
  Claude trend-analysis job.

All historical analytics are clearly marked synthetic and live under
`src/demo-data/trends/`. They are never sent to MinIO or consumed by the Go
services. The current-day values match the selected synthetic repository
fixtures; the prior six days are deterministic demo history.

## Validate

Run the demo catalog unit tests, validate the trend fixture, and build the UI:

```shell
npm run check:catalog
npm run check:trends
npm run build
```

From the original repository layout, regenerate it with:

```shell
npm run generate:trends
npm run check:trends
```

The checked-in fixture is self-contained when `demo-ui/` is copied to a separate
work-laptop folder. Regeneration is optional.
