# Read-only coverage addendum — 2026-10-31

This addendum records the current iteration's bounded verification.

## Native read-only additions

- `cloudsigma_usage`: GET `/api/2.0/usage/` with `poll_time_gt`, `poll_time_lt`, and `limit`.
- `cloudsigma_balance`: GET `/api/2.0/balance/`.
- `cloudsigma_burstusage`: GET `/api/2.0/burstusage/` with date filters.
- `cloudsigma_dailyburstusage`: GET `/api/2.0/dailyburstusage/` with date filters.
- `cloudsigma_pricing`: GET `/api/2.0/pricing/` with optional `limit`.
- `cloudsigma_discount`: GET `/api/2.0/discount/`.

All preserve dynamic JSON and do not infer pagination completeness or lifecycle semantics.

## Verification

- `go test ./internal/cloudsigmaapi -count=1` — PASS.
- `go test ./internal/provider -run '^TestOffline' -count=1` — PASS.
- `go test -race ./internal/provider -run '^TestOffline' -count=1` — PASS.
- `go test -race ./internal/cloudsigmaapi -run 'Usage|Balance|Burst|Pricing|Discount|SnapshotWrite|Scheduler' -count=1` — PASS.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `make lint` — PASS, 0 issues.
- `make docs` — PASS.
- `git diff --check` — PASS.

The broad `go test ./internal/provider -count=1` and race equivalent were each stopped after about 60 seconds with no output while the existing long Native28 lifecycle matrix was running. They are retained as bounded-suite limitations, not relabeled as pass or failure.
