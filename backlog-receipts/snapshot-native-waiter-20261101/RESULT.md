# Native snapshot waiter — 2026-11-01

Added the provider-owned snapshot waiter seam needed before removing the SDK
from snapshot lifecycle operations.

- `Client.WaitSnapshotAvailable` polls native `GET /snapshots/{uuid}/`.
- It preserves the existing pending statuses `creating` and `cloning_dst`,
  accepts only `available` as success, and rejects terminal or unknown states.
- Default timing remains the existing provider behavior: ten-minute timeout
  and three-second delay. Caller cancellation/deadlines take precedence.
- Snapshot resource create uses the native waiter when a native client is
  configured and retains the legacy waiter for legacy-only configurations.
- The existing final native snapshot read remains after the waiter, preserving
  state projection and request ordering.
- The waiter checks already-cancelled/deadline contexts before each native
  read; existing `SnapshotWithCallerContext` behavior remains unchanged.

Validation:

- `go test ./internal/cloudsigmaapi -run 'SnapshotCallerContext|WaitSnapshotAvailable|SnapshotWrite' -count=1` — passed.
- `go test -race ./internal/cloudsigmaapi -run 'SnapshotCallerContext|WaitSnapshotAvailable|SnapshotWrite' -count=1` — passed.
- `go test ./internal/provider -run 'TestNative28SnapshotLifecycle' -count=1` — passed.
- `go test -race ./internal/provider -run 'TestNative28SnapshotLifecycle' -count=1` — passed.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed.

This slice does not switch snapshot create/update/delete to native transport.
Those operations still require state/import and full lifecycle parity review.

## Deadline-policy correction

Native refreshes keep the caller's request context unchanged. The waiter owns
its overall timeout with an external timer, matching the legacy waiter's
deadline policy and preserving the Native28 `deadline-policy` contract.

Focused verification: `TestNative28Matrix/credentials3/rawfalse/deadline-policy`
and `rawtrue` both pass after this correction.
