# Native audit-log read — 2026-11-01

Implemented provider-owned read-only coverage for the documented
`GET /api/2.0/logs/` audit-log collection.

- Added a bounded native `logs/` route with optional `limit` query projection.
- Preserved deployment-specific log fields as dynamic JSON.
- Marked the Terraform `json` attribute sensitive because documented audit
  details may contain credential-like request values.
- Registered `cloudsigma_audit_logs` as a read-only Terraform data source.
- Added Basic/bearer transport coverage, offline provider coverage, schema and
  registration updates, generated documentation, and no mutation semantics.

Validation:

- targeted normal and race tests — passed;
- `go vet ./internal/cloudsigmaapi ./internal/provider` — passed;
- `go build ./...` — passed;
- `make docs` — passed;
- `git diff --check` — passed.

The offline semantic suite includes `TestOfflineAuditLogsDataSource`; its
provider event count is qualified at 460. The fixture uses the in-memory
transport seam used by the offline isolation suite and opens no listener.

No live API calls or account mutations were performed.
