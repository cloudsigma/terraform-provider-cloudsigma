# Native scheduler assignment qualification — 2026-10-31

## Scope

Added provider-owned library methods for the source-reviewed drive scheduler
actions. This slice does not register a Terraform resource and does not infer
backup completion, exactly-once behavior, or cross-replica read-after-write
consistency.

## Implementation

- `AssignDriveScheduler` sends `POST /api/2.0/drives/{uuid}/action/?do=set_scheduler`.
- `UnassignDriveScheduler` sends the corresponding `remove_scheduler` action.
- Both send only `{"backup_policy_uuid":"..."}`.
- Both use the configured Basic or bearer authentication mode.
- Only HTTP `202 Accepted` is admitted.
- The returned drive acknowledgement is decoded and validated; malformed,
  redirected, unexpected-status, transport, timeout, and cancellation outcomes
  are surfaced without replay.
- `Drive.BackupScheduler` captures the optional readback link.

## Validation

- `GOPROXY=off go test ./internal/cloudsigmaapi -run 'TestDriveScheduler|TestCreateBackupScheduler|TestCreateRemoteSnapshot' -count=1 -timeout=120s` — PASS
- `GOPROXY=off go test -race ./internal/cloudsigmaapi -run 'TestDriveScheduler|TestCreateBackupScheduler|TestCreateRemoteSnapshot' -count=1 -timeout=120s` — PASS
- `GOPROXY=off go test ./internal/cloudsigmaapi -count=1 -timeout=240s` — PASS
- `GOPROXY=off go vet ./internal/cloudsigmaapi ./internal/provider` — PASS
- `GOPROXY=off go build ./...` — PASS
- `git diff --check` — PASS

## Remaining qualification boundary

Issue #30 remains open for a Terraform assignment resource until the API
documents read-after-write consistency, timeout/failure/cancellation behavior,
concurrent assignment semantics, import/ownership, and absent-remove rules.
