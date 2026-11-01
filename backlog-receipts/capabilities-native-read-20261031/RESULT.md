# Native capabilities read qualification — 2026-10-31

Added provider-owned `Client.Capabilities` for the documented
`GET /api/2.0/capabilities/?limit=0` endpoint. The method preserves the dynamic
document as `map[string]json.RawMessage`, supports Basic and bearer auth, and
does not hard-code deployment limits or perform hidden plan-time validation.

Registered an explicit `cloudsigma_capabilities` data source exposing the
canonical JSON document and stable `capabilities` identity. It performs a read
only request during normal data-source evaluation; schema validation remains
network-free.

Synthetic tests cover the exact route/query, dynamic nested fields, bearer
authentication, redirects, malformed/null responses, and transport failure.

Validation:

- `GOPROXY=off go test ./internal/cloudsigmaapi -run '^TestCapabilities' -count=1 -timeout=120s` — PASS
- `GOPROXY=off go test ./internal/cloudsigmaapi -count=1 -timeout=240s` — PASS
- `GOPROXY=off go test -race ./internal/cloudsigmaapi -count=1 -timeout=300s` — PASS
- `GOPROXY=off go test ./internal/provider -run 'TestOfflineCapabilitiesDataSource|TestOfflineRegisteredSchemas' -count=1 -timeout=180s` — PASS
- `GOPROXY=off go test -race ./internal/provider -run 'TestOfflineCapabilitiesDataSource' -count=1 -timeout=180s` — PASS
