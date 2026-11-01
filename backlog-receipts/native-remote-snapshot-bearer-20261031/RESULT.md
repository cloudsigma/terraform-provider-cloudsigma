# Remote Snapshot native bearer parity — 2026-10-31

The preserved native Remote Snapshot create transport now calls the provider
client authentication selector, so Basic and bearer clients use their configured
mode. A synthetic test verifies bearer requests emit `Authorization: Bearer`
and no Basic credentials.

Validation:

- `GOPROXY=off go test ./internal/cloudsigmaapi -run 'TestCreateRemoteSnapshot' -count=1 -timeout=120s` — PASS
- `GOPROXY=off go test -race ./internal/cloudsigmaapi -run 'TestCreateRemoteSnapshot' -count=1 -timeout=120s` — PASS

This does not qualify Remote Snapshot managed CRUD. Durable deletion completion
and reappearance behavior remain unresolved; the draft resource stays
unregistered.
