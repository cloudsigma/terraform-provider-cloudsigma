# API documentation and surface-gap review — 2026-10-31

This receipt records a read-only review of the public CloudSigma API v2 documentation and unauthenticated `OPTIONS` probes against `next.cloudsigma.com`. No credentials, resource mutations, or live acceptance calls were used.

## Contract observations

- Snapshots expose `GET`, `POST`, `PUT`, and `DELETE`. The documented create response is HTTP 201 with an `objects` array; update is HTTP 200; delete is HTTP 204 and explicitly says deletion started. Snapshot clone is an asynchronous HTTP 202 action whose response is a drive definition and may include a jobs link.
- Remote snapshots expose the same verbs. The docs explicitly prohibit edit, delete, and clone while a remote snapshot is `creating` or `transferring`, and document a per-drive creation interval. Delete remains an accepted asynchronous start, not a durable terminal-success contract.
- The remote-snapshot page is internally inconsistent: it labels the resource as `/remotesnapshots/` but several examples use `/api/2.0/snapshots/`. Those examples are not sufficient authority for native request routing or writable-field projection.
- Backup scheduler field reference separates required top-level fields (`type`, `name`, enable/default flags, notification flags, retention policy, incremental backup, user timezone) from optional fields (`manual_incremental_backup`, `initial_backup`, remote location, metadata, tags). Examples demonstrate nested time-window and repeat grammar, but assignment remains an asynchronous 202 action without a documented terminal completion, failure, unassignment, or reconciliation guarantee.
- The documented read-only surfaces not yet represented by provider-owned native transport include `currentusage`, `usage`, `balance`, `burstusage`, and billing/discount. Notification contacts/preferences, ACLs, virtual routers and nested features, and upload/download sessions are additional API surfaces requiring separate contract work.

## Sources

- https://docs.cloudsigma.com/en/latest/snapshots.html
- https://docs.cloudsigma.com/en/latest/remote_snapshots.html
- https://docs.cloudsigma.com/en/latest/backup_schedulers.html
- https://docs.cloudsigma.com/en/latest/backup_schedulers_field_reference.html
- https://docs.cloudsigma.com/en/latest/current_usage.html
- https://docs.cloudsigma.com/en/latest/usage.html
- https://docs.cloudsigma.com/en/latest/billing.html
- https://docs.cloudsigma.com/en/latest/notification_preferences.html
- https://docs.cloudsigma.com/en/latest/acls.html
- https://docs.cloudsigma.com/en/latest/upload_download.html

## OPTIONS probe

Command used:

```sh
for p in backupschedulers snapshots remotesnapshots currentusage usage balance burstusage notification_preferences acls virtual_routers; do
  curl -sS -D - -o /tmp/curlbody -X OPTIONS --max-time 10 "https://next.cloudsigma.com/api/2.0/$p/"
done
```

Observed `Allow` headers:

| Endpoint | Allow |
|---|---|
| `backupschedulers/` | `GET,POST,DELETE` |
| `snapshots/` | `GET,PUT,DELETE,POST` |
| `remotesnapshots/` | `GET,PUT,DELETE,POST` |
| `currentusage/` | `GET` |
| `usage/` | `GET` |
| `balance/` | `GET` |
| `burstusage/` | `GET` |
| `notification_preferences/` | `GET,PUT,DELETE,POST` |
| `acls/` | `GET,PUT,POST,DELETE` |
| `virtual_routers/` | `GET,PUT,DELETE,POST` |

These probes are capability hints only; they do not establish request schemas, response guarantees, authentication parity, or lifecycle safety.
