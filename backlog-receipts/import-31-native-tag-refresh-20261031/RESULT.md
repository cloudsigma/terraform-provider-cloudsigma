# Native tag import refresh qualification — 2026-10-31

Added offline compatibility coverage for the existing passthrough importer:

- imported ID-only `cloudsigma_tag` state refreshes through the provider-owned
  native `GET /api/2.0/tags/{uuid}/` path;
- Basic authentication is selected by the configured native client;
- the refresh populates the imported tag name and resource URI;
- a native `404` removes the imported resource from state;
- no create, SDK fallback, or live API operation is involved.

Validation:

- `GOPROXY=off go test ./internal/provider -run '^TestOfflineTagResourceImportRead' -count=1 -timeout=120s` — PASS
- matching `-race` command — PASS
- `git diff --check` — PASS
