# Native scheduler read data source — 2026-10-31

Added the explicit read-only `cloudsigma_backup_schedulers` data source backed
by native `GET /api/2.0/backupschedulers/?limit=0`.

- Preserves the complete scheduler response objects as canonical JSON.
- Rejects incomplete pagination metadata, malformed objects, redirects, and
  trailing JSON in the native transport.
- Supports Basic and bearer client modes.
- Does not register scheduler create/update/delete or drive-assignment lifecycle.

Validation:

- Offline data-source, schema-registration, and coverage tests — PASS.
- Native scheduler transport tests — PASS.
