# Native SSH key token parity — 2026-10-31

The registered SSH key resource now selects the provider-owned native keypair
transport for both Basic and bearer authentication. Token configurations no
longer fall back to the Go SDK for SSH key Read/Create/Update/Delete callbacks.

The existing schema and import identity are unchanged. Offline fixtures now
exercise the native path for both auth modes while preserving response/status
and state-boundary assertions.

Validation:

- `GOPROXY=off go test ./internal/provider -run '^TestOfflineNativeSSHKey$' -count=1 -timeout=120s` — PASS
- `GOPROXY=off go test ./internal/provider -run 'TestOfflineNativeSSHKey(Delete|Update|StateBoundaries|DeleteInvalid|UpdateInvalid)$' -count=1 -timeout=180s` — PASS
