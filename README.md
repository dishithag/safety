# internal

Private application code. Go's `internal/` rule prevents other modules from
importing anything here, so this is where the bulk of the implementation lives.

## Layout

- `common/` — infrastructure shared across services (data-store clients, wiring).
- `<service>/` — one directory per service, split into layers:
  - `domain/` — service-specific entities, interfaces, and config (no I/O).
  - `usecase/` — application logic; orchestrates domain objects via interfaces.
  - `gateways/` — adapters to the outside world (HTTP API, message bus, etc.).
  - `validation/` — request/input validation rules.

Dependencies point inward: `gateways` and `usecase` depend on `domain`
interfaces, never the reverse. Add a `README.md` per service documenting owners,
infrastructure, and dependencies.

Implemented services:

- [`summarizer/`](summarizer/README.md) — report parsing, deterministic analysis,
  GenAI Hub integration, validation, rendering, freshness, and MinIO/S3 access.
- [`analyticsapi/`](analyticsapi/README.md) — S3-backed narrative retrieval and
  HTTP delivery.
