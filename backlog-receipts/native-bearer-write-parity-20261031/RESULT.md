# Native bearer write parity audit — 2026-10-31

All provider-owned native write call sites now use `Client.authenticate`.
Hard-coded Basic authentication was removed from drive, server, snapshot,
keypair, attachment, clone, create, delete, and power methods. The only
remaining `SetBasicAuth` call is inside the authentication helper itself.

An explicit bearer-auth synthetic test covers keypair creation. Existing Basic
tests remain unchanged, and no SDK fallback or resource registration was
introduced.

Validation:

- `GOPROXY=off go test ./internal/cloudsigmaapi -count=1 -timeout=240s` — PASS
- `GOPROXY=off go test ./internal/cloudsigmaapi -run 'TestKeypairCreate' -count=1 -timeout=120s` — PASS
- `git diff --check` — PASS
