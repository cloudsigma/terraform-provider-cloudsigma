# Native SSH-key resource SDK removal — 2026-11-01

The registered `cloudsigma_ssh_key` resource now requires the provider-owned native client for both Basic and bearer configurations.

- Removed the resource's runtime SDK fallback for create, read, update, and delete.
- Preserved existing state projection and import behavior.
- Kept native request projections and response identity checks.
- Added no new lifecycle semantics; the existing keypair API contract is used.

Validation:

- Focused SSH-key and provider configuration tests passed.
- Full native API tests passed.
- Full provider offline tests passed normally and under race.
- `go vet ./...`, `go build ./...`, `make lint`, `make docs`, and `git diff --check` passed.

The provider construction and legacy drive/server mux still retain SDK dependencies. Snapshot resource mutation routing remains SDK-backed pending native waiter/state qualification.
