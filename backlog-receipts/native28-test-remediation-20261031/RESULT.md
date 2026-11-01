# Native28 test remediation — 2026-10-31

## Local fixes

- `native28_snapshot_lifecycle_test.go` now gives each synthetic fault case a
  bounded 15-second caller context. Production snapshot waiter timeout remains
  ten minutes.
- `native28_matrix_test.go` synchronizes shared fixture counters/order and
  snapshots state before assertions. Blocked-body cleanup is awaited before
  assertions, and the deadline/cancellation boundary no longer races on the
  shared fixture state.
- `library_drives_test.go` uses a 50ms synthetic deadline instead of 1ms to
  avoid full-suite scheduling flakiness while preserving the deadline contract.

## Passing validation

- `TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform GOPROXY=off go test ./internal/provider -run '^TestNative28SnapshotLifecycle$' -count=1 -timeout=5m` — PASS (112s)
- Matching `-race` command — PASS (114s)
- Native28 deadline/final-deadline subset under `-race` — PASS (18s)
- Native28 success/pending/cancel/malformed subset under `-race` — PASS (43s)
- Real-CLI provider/unit suite and matching race suite — PASS
- Full native API tests and race tests — PASS
- Linux amd64 test binaries for `internal/cloudsigmaapi` and `internal/provider` compile successfully.
- Vet, build, lint (`0 issues`), and diff check — PASS

## Remaining boundary

The complete Native28 matrix remains a large synthetic stress matrix and was
not claimed as a single full-package pass. Linux execution is now secondary
portability validation; local deterministic test defects were fixed first.
