# Native Jobs data source — 2026-10-31

Registered `cloudsigma_jobs` as an explicit read-only Framework data source.
It uses the native `GET /api/2.0/jobs/?limit=0` transport, preserves recursive
child jobs and dynamic progress data as canonical JSON, and performs no
mutation, polling, or lifecycle inference.

Validation:

- Offline Jobs data-source and registration/coverage tests — PASS
- Full offline provider suite and race suite — PASS
- Native Jobs transport tests — PASS
- `go vet ./...`, `go build ./...`, `make lint`, `git diff --check` — PASS
