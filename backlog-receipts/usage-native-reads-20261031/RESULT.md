# Native usage and billing reads — 2026-10-31

Implemented bounded provider-owned read-only transport and JSON data sources:

- `GET /api/2.0/usage/` with `poll_time_gt`, `poll_time_lt`, and `limit` query fields;
- `GET /api/2.0/balance/`;
- `GET /api/2.0/burstusage/` with date filter fields;
- `GET /api/2.0/dailyburstusage/` with date filter fields.

Responses remain dynamic JSON because the API documents usage and burst objects with deployment-specific resource fields and inconsistent pagination envelopes (`burstusage` examples omit `meta`, while `dailyburstusage` includes it). No pagination completeness or lifecycle semantics are inferred.

Registered data sources:

- `cloudsigma_usage`
- `cloudsigma_balance`
- `cloudsigma_burstusage`
- `cloudsigma_dailyburstusage`

Validation:

- Native usage/balance/burst/daily tests passed normally and under race.
- Offline provider data-source tests passed normally and under race.
- `go vet ./...`, `go build ./...`, `git diff --check`, and `make docs` passed.
- Generated docs are present under `docs/data-sources/`.

Sources reviewed:

- https://docs.cloudsigma.com/en/latest/usage.html
- https://docs.cloudsigma.com/en/latest/billing.html
- https://docs.cloudsigma.com/en/latest/burst_usage.html
- https://docs.cloudsigma.com/en/latest/current_usage.html
