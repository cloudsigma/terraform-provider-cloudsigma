# Native SSH-key bearer parity — 2026-10-31

The Framework SSH-key resource now selects the provider-owned native client for
both Basic and bearer configurations. Its existing native Keypair CRUD methods
already use the selected authentication mode; the prior token configuration
left the resource on the SDK path.

Offline token fixtures now exercise native read, update, and delete status and
projection behavior. Legacy schema/import compatibility remains unchanged.

Validation:

- Focused native SSH-key tests — PASS
- Matching focused race tests — PASS
- Full offline provider suite and race suite — PASS
- `go vet ./...`, `go build ./...`, `make lint`, `git diff --check` — PASS
