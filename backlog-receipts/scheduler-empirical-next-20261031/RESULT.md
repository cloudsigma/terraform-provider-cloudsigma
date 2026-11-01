# Empirical scheduler contract qualification — 2026-10-31

## Test account and cleanup

- Account: 1Password `Clawdbot / Next QA account`.
- Endpoint: `https://next.cloudsigma.com`.
- All calls were serial and limited to one uniquely named scheduler,
  `codex-audit-20261031-1906`.
- The scheduler was updated and then deleted by its exact UUID. Immediate detail
  GET returned `404`, and the collection no longer contained the UUID.
- No drive, server, snapshot, or other billable resource was created.

## Observed API behavior

### Scheduler collection/detail

- `OPTIONS /api/2.0/backupschedulers/` returned `Allow: GET,POST,DELETE`.
- `OPTIONS /api/2.0/backupschedulers/{uuid}/` returned `Allow: GET,PUT,DELETE`.
- Empty collection returned HTTP `200` with `meta.limit=1000`, `meta.offset=0`,
  `meta.total_count=0`, and `objects=[]`.

### Create

- A POST with `manual_incremental_backup` omitted returned HTTP `400`:
  `error_point=manual_incremental_backup`, `Field manual_incremental_backup cannot be empty`.
- A valid minimal writable projection with `manual_incremental_backup="00:00"`
  returned HTTP `201` with an `objects` response and UUID
  `1350cf9a-ea96-49fd-b9d4-293d07385dcc`.
- Submitted `meta` and `tags=[]` were accepted. The response included generated
  owner/permissions/grantees and normalized `initial_backup={}` and empty
  timezone/location fields.

### Update

- A minimal PUT containing only `name`, `meta`, and `tags` returned HTTP `400`:
  `error_point=incremental_backup`, `Field incremental_backup cannot be empty`.
- A full writable scheduler projection was required. The first full PUT then
  returned HTTP `400` because `initial_backup.day_of_week` was empty.
- Adding a populated `initial_backup` block made the PUT return HTTP `200` and
  changed both `name` and `meta`.
- This confirms update requires a complete nested writable projection; it is not
  a patch-style update.

### Delete

- DELETE returned HTTP `204` with an empty body.
- Immediate detail GET returned HTTP `404` and the subsequent collection omitted
  the UUID. This establishes observed next-environment behavior, but does not
  prove a universal no-reappearance guarantee after asynchronous failure.

### Assignment probes

- `set_scheduler` and `remove_scheduler` against a nonexistent drive both
  returned HTTP `404` with `error_type=notexist` and the drive UUID in the error.
- No suitable existing drive was present in the QA account, so no real
  drive-to-scheduler assignment was attempted.

## Provider implications

- The native create validator now requires nonempty
  `manual_incremental_backup`, matching observed server behavior.
- Scheduler create remains library-only and scheduler update/delete remain
  unregistered Terraform lifecycles until cross-deployment semantics,
  complete nested projection, assignment readback, and durable deletion
  guarantees are established.
