# Native scheduler resources — 2026-11-01

DevOps supplied the pinned Turlo revision for `next.cloudsigma.com`:
`f42d7a8d381805c9d397cec44e50269171f59679`.

This slice implements the smallest provider-owned Terraform lifecycle directly
qualified by that contract:

- `cloudsigma_scheduler` owns scheduler CRUD using the complete writable JSON
  projection; response-only fields are removed from update state.
- `cloudsigma_scheduler_assignment` exclusively owns the drive-to-scheduler
  association. It reconciles through `drive.backup_scheduler`, serializes
  mutations per process, reads before retrying uncertain actions, and refuses
  to replace a different concurrent association during refresh.
- Scheduler CRUD accepts the pinned deployment's types and clearing values;
  full PUT semantics are preserved rather than using PATCH behavior.
- Import is identifier-only and refresh reads the authoritative detail/drive
  endpoints.

Validation:

- scheduler native transport tests, resource lifecycle tests, assignment
  reconciliation tests — passed;
- matching race tests — passed;
- provider and API vet/build — passed;
- offline provider suite includes both resources and is qualified at 460 events;
- no live API, account, or billable operation was performed.

The generated offline guard remains platform-specific and is addressed by the
separate lint portability commit; no RemoteSnapshot lifecycle semantics are
introduced here.
