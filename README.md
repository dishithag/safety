# Zero Trust Analytics Demo UI

Standalone frontend for browsing generated per-CID Markdown narratives. This
folder is intentionally isolated from the Go services and is not part of the
production deployment.

## Run locally

Start the existing analytics API on `http://localhost:8080`, then run:

```shell
cd demo-ui
npm install
npm run dev
```

Open `http://localhost:5173`. Vite proxies narrative requests to the existing
analytics API, so no backend CORS change is required.

To use another API origin, create `.env.local`:

```text
VITE_API_BASE_URL=http://localhost:8080
```

Direct cross-origin requests require that API origin to permit browser CORS.

## Scope

The current UI deliberately supports only the report-demo workflow:

- Search the curated sample CID list.
- Fetch one narrative through
  `GET /zero-trust-analytics/narratives/{cid}`.
- Render the Markdown with readable headings, tables, lists, and navigation.
- Show loading, missing-report, and API-error states.

Trend analytics and synthetic historical data are not included in this first
milestone.
