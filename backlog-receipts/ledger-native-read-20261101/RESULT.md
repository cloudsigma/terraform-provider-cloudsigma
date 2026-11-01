# Native ledger read — 2026-11-01

Implemented provider-owned read-only ledger coverage from the documented
`GET /api/2.0/ledger/` transaction list:

- Added a closed native `ledgerEndpoint` route and `Client.Ledger` method.
- Preserved the paginated, deployment-specific response as dynamic JSON.
- Registered `cloudsigma_ledger` as a read-only Terraform data source.
- Added Basic/bearer transport coverage, synthetic provider coverage, schema
  registration updates, and generated documentation.
- No mutation or Terraform ownership semantics were added.

Validation:

- `go test ./internal/cloudsigmaapi ./internal/provider -run 'Ledger|Usage|RegisteredSchemas|DatasourceCoverage' -count=1` — passed.
- `go test -race ./internal/cloudsigmaapi ./internal/provider -run 'Ledger|Usage|RegisteredSchemas|DatasourceCoverage' -count=1` — passed.
- `go vet ./internal/cloudsigmaapi ./internal/provider` — passed.
- `go build ./...` — passed.
- `make docs` — passed; generated `docs/data-sources/ledger.md`.
- `git diff --check` — passed.

The repository-wide `make lint` remains blocked by a pre-existing
`build/offline/internal-cloudsigmaapi_guard_test.go` Darwin typecheck failure:
`undefined: syscall.SockFilter`, `undefined: syscall.SockFprog`, and
`undefined: syscall.SYS_PRCTL`. The new ledger packages pass targeted checks;
the blocker was not relabeled as a ledger failure.
