# Snapshot lifecycle reassessment after native coverage merge — 2026-11-01

PR #146 is merged on GitHub at `55525e0ecc4ab2228780594fbecb14c12d78451b`.
The local `origin/main` ref is stale at the pre-merge base `160845a`; this
reassessment uses the merged PR head `a1ac4172e60365b33d258ef7661d8f345097e09f`.

## Current qualified pieces

- Native read: `GET /api/2.0/snapshots/{uuid}/` with Basic and bearer auth.
- Native create transport: `POST /api/2.0/snapshots/`, minimal `{drive:{uuid},name}` projection, strict `201`, one acknowledged UUID, redirect refusal, bounded context, and response decoding tests.
- Native update transport: `PUT /api/2.0/snapshots/{uuid}/`, minimal writable projection, strict `200`, matching response UUID, and failure/context tests.
- Native waiter: `creating` and `cloning_dst` are pending; `available` is success; terminal/unknown states fail; existing ten-minute/three-second timing is preserved without changing caller request deadlines.
- Native delete transport: strict `204` acknowledgement remains the current behavior.

The snapshot resource currently uses native read/wait/delete only when configured for Basic auth. Create and update still dispatch through the SDK. Bearer configurations still select the SDK snapshot paths because resource configuration has not yet enabled native snapshot lifecycle routing for token auth.

## Smallest qualified next implementation

Wire `snapshotResource.Create` to `r.native.CreateSnapshot` for native Basic configurations. Keep the existing drive preflight, native waiter, final native read, state projection, and legacy path unchanged. Update the Native28 fake-transport expectations from legacy POST to native POST for the native branch and retain exact identity/unknown-outcome/error checks.

The native POST contract is already covered by `snapshot_write_test.go`, the public snapshot documentation, and the pinned native coverage evidence. No live mutation is required to qualify this bounded routing change.

## Follow-on

Wire `snapshotResource.Update` to `r.native.UpdateSnapshot` for native Basic after adding resource-level state/error tests. Then qualify bearer routing separately. Full SDK removal remains blocked by SDK-typed provider construction, drive/server legacy mux, and SDK waiter/preflight signatures.

## Evidence

- `internal/cloudsigmaapi/snapshots.go`
- `internal/cloudsigmaapi/snapshot_write_test.go`
- `internal/cloudsigmaapi/snapshot_wait.go`
- `internal/cloudsigmaapi/snapshot_wait_test.go`
- `internal/provider/resource_cloudsigma_snapshot.go`
- `internal/provider/native28_snapshot_lifecycle_test.go`
- `internal/provider/native28_matrix_test.go`
- `backlog-receipts/snapshot-native-write-20261031/RESULT.md`
- `backlog-receipts/snapshot-native-waiter-20261101/RESULT.md`
- Public docs: https://docs.cloudsigma.com/en/latest/snapshots.html
