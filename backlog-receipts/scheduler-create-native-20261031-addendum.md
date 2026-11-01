# Scheduler create contract addendum — 2026-10-31

The native scheduler create projection was narrowed to the source-reviewed
field shapes:

- tags use minimal UUID references;
- `orchestrator_timezone` is represented when supplied;
- metadata keys are limited to 32 characters and values are strings or booleans;
- `manual_incremental_backup` is passed through to the server so wildcard and
  comma-separated forms are not rejected by an over-strict client parser.

Synthetic tests cover projection, invalid tag/metadata values, and a wildcard
plus comma-separated manual backup value. The scheduler remains unregistered;
canonical update and assignment lifecycle semantics are still open.
