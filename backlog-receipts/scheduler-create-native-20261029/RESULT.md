# Native scheduler create qualification — 2026-10-29

## Contract basis

The implementation uses the public Backup Scheduler field reference and the
documented PUT request shape as the canonical writable projection, because the
POST request code block is blank while its response contract is explicit.

- Required projection: `type`, `name`, `is_enabled`, `is_default`, both
  notification flags, `retention_policy`, `incremental_backup`, and
  `user_timezone`.
- Optional projection: `manual_incremental_backup`, `initial_backup`,
  `remote_location`, and `meta`.
- `month` accepts the documented string or string-list forms.
- `POST /backupschedulers/` requires HTTP 201 and one acknowledged UUID.
- No drive assignment, polling, or completion inference is included.

Sources:

- https://docs.cloudsigma.com/en/latest/backup_schedulers.html
- https://docs.cloudsigma.com/en/latest/backup_schedulers_field_reference.html

## Implementation

Added provider-owned `Client.CreateBackupScheduler` and typed scheduler request
models in `internal/cloudsigmaapi/backup_schedulers.go`. The request emits an
`objects` envelope containing only writable fields. Response-only identity and
ownership fields cannot enter the request model.

## Verification

- `GOPROXY=off go test ./internal/cloudsigmaapi -run 'BackupScheduler|CreateBackupScheduler' -count=1 -timeout=120s` — PASS
- `GOPROXY=off go test -race ./internal/cloudsigmaapi -run 'BackupScheduler|CreateBackupScheduler' -count=1 -timeout=180s` — PASS
- `GOPROXY=off go test ./internal/provider -run '^TestOffline' -count=1 -timeout=180s` — PASS
- `GOPROXY=off go test ./internal/cloudsigmaapi -count=1 -timeout=180s` — PASS
- `GOPROXY=off go vet ./internal/cloudsigmaapi ./internal/provider` — PASS
- `GOPROXY=off go build ./...` — PASS
- `git diff --check` — PASS

## Remaining issue #30 scope

The drive assignment endpoint remains separate: `POST
/drives/{drive_uuid}/action/?do=set_scheduler` is documented only as HTTP 202
accepted. Completion, failure, timeout, read-after-assignment, and unassign
semantics remain undocumented. Optional scheduler `tags` element shape is also
not included in the typed create projection.

No live cloud, QA account, credential, acceptance, release, deployment, or
public repository write was performed for this local implementation.
