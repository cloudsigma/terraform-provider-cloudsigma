# Native pricing and discount reads — 2026-10-31

Added provider-owned, read-only dynamic JSON transport and data sources for:

- `GET /api/2.0/pricing/`, with an optional `limit` query field;
- `GET /api/2.0/discount/`.

The response remains dynamic JSON because pricing contains deployment-specific
resource rows and current/next burst maps. No anonymous response was treated as
an authentication contract; native Basic/bearer authentication is always used.

Registered data sources:

- `cloudsigma_pricing`
- `cloudsigma_discount`

Validation:

- Native normal and race tests cover route/query projection, Basic/bearer auth,
  response decoding, and existing dynamic status/trailing/transport failures.
- Provider offline normal and race tests cover both data sources.
- `go vet ./...`, `go build ./...`, `make lint`, `make docs`, and
  `git diff --check` pass.
- Read-only OPTIONS probes on `next.cloudsigma.com` returned `200 Allow: GET`
  for both endpoints; no credentials or mutations were used.
