# Native audit-log read — 2026-11-01

Implemented provider-owned read-only coverage for the documented
`GET /api/2.0/logs/` audit-log collection.

- Added a bounded native `logs/` route with optional `limit` query projection.
- Preserved deployment-specific log fields as dynamic JSON.
- Registered `cloudsigma_audit_logs` as a read-only Terraform data source.
- Added Basic/bearer transport coverage, offline provider coverage, schema and
  registration updates, generated documentation, and no mutation semantics.

Validation:

- targeted normal and race tests — passed;
- `go vet ./internal/cloudsigmaapi ./internal/provider` — passed;
- `go build ./...` — passed;
- `make docs` — passed;
- `git diff --check` — passed.

No live API calls or account mutations were performed.
