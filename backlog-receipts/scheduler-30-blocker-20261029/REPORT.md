# Scheduler issue #30 — contract status update (2026-10-29)

Issue #30 remains open because the public API now documents a scheduler field reference, but it still does not provide a safe Terraform lifecycle contract.

## Confirmed public surface

- `GET /backupschedulers/`, detailed list variants, and `GET /backupschedulers/{uuid}/`.
- `POST /backupschedulers/` returns `201` with an `objects` response, but the published create request example is empty.
- `PUT /backupschedulers/{uuid}/` returns `200` and shows a full nested scheduler object.
- `DELETE /backupschedulers/{uuid}/` returns `204` and says deletion started.
- `POST /drives/{drive_uuid}/action/?do=set_scheduler` accepts `backup_policy_uuid` and returns `202` with a `backup_scheduler` link.
- The field reference identifies required top-level fields and allowed scheduler types (`snapshot` and `backup`).

Sources:

- https://docs.cloudsigma.com/en/latest/backup_schedulers.html
- https://docs.cloudsigma.com/en/latest/backup_schedulers_field_reference.html

## Why implementation is still blocked

The scheduler create request is still not normatively specified. Nested initial/incremental/retention structures contain multiple string, number, list, and optional forms, and the docs do not establish the complete writable projection or omission/null/empty semantics. The drive assignment endpoint is explicitly asynchronous (`202`) without a documented completion/failure/timeout contract, read-after-assignment guarantee, or unassign operation. A Terraform resource or drive assignment field would therefore claim lifecycle behavior the public contract does not establish.

## Smallest enabling documentation/API extension

Publish one canonical create/update request schema with writable and response-only fields, nested grammar, accepted values, and omission/null semantics. Document assignment completion, failure and timeout behavior, read-after-assignment state, and unassignment. Define scheduler import/read identity and deletion semantics for Terraform.

No speculative scheduler implementation was added. Existing blocker evidence remains preserved.
