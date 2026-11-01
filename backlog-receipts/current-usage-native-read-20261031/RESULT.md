# Native current usage read — 2026-10-31

Implemented provider-owned read-only current usage coverage:

- `GET /api/2.0/currentusage/`
- Dynamic JSON model preserving deployment-specific `balance` and `usage` fields.
- Registered `cloudsigma_current_usage` data source with stable `currentusage` identity.
- Basic and bearer authentication through the native client.

Validation:

- `go test ./internal/cloudsigmaapi ./internal/provider -run 'CurrentUsage|current_usage' -count=1` — passed.
- `go test -race ./internal/cloudsigmaapi ./internal/provider -run 'CurrentUsage|current_usage' -count=1` — passed.
- `go test ./internal/provider -run '^TestOffline' -count=1` — passed after registration count update.
- `go test -race ./internal/provider -run '^TestOffline' -count=1` — passed.
- `go test ./internal/cloudsigmaapi -count=1` — passed.
- `go vet ./...`, `go build ./...`, `make lint`, and `git diff --check` — passed.
- `make docs` — passed using the cached `tfplugindocs` tool and local Terraform CLI; generated current usage, capabilities, jobs, and scheduler data-source documentation.

The full unfiltered `go test ./... -count=1` run was stopped after more than two minutes without additional output while the existing broad Native28 stress suite was running. A second broad run with `-skip 'Native28|SnapshotLifecycle'` was also stopped after roughly 2.5 minutes without output. These are retained as bounded-suite limitations, not relabeled as passes or failures of the new current-usage slice.
