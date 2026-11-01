# Native Remote Snapshot Migration

Base: e8c6938ace29cd865a5e64b26f6e73b4ba55ca24
Working tree: uncommitted bounded candidate; exact base `e8c6938ace29cd865a5e64b26f6e73b4ba55ca24`; current tree `08fb3c68dc96d2e9d79418bb26568e3d6f6f875a`

## Actual changes
- Added provider-owned internal/cloudsigmaapi remote snapshot request/response models and POST transport.
- Added native-only Terraform resource registration and create/state adapter.
- Removed SDK dependency from remote snapshot resource candidate; retained existing candidate paths by replacing stale draft behavior.
- Added offline synthetic HTTP create test.

## Contract boundary
Create uses POST /api/2.0/remotesnapshots/ with one-object objects batch, drive UUID and optional location, requires 201 and exactly one usable UUID. Response-only fields are projected into state. No polling is performed.
Read, update, and delete remain explicit diagnostics because no authoritative semantics were available; this is not a complete Terraform CRUD resource.

## Verification
- Focused offline verification at 2026-10-26T15:04:08Z passed: `go test ./internal/cloudsigmaapi ./internal/provider -run 'TestOfflineRemoteSnapshotNativeCreate|^$' -count=1`.
- `git diff --check` and `gofmt -d` were clean in the earlier 14:29Z receipt and the candidate tree is unchanged at `08fb3c68dc96d2e9d79418bb26568e3d6f6f875a`.
- Repository-wide `go test ./... -count=1 -timeout=45s` was attempted and failed by the configured test timeout, not by the RemoteSnapshot test: existing live/acceptance-style tests `TestNative26AttachmentFaults` and `TestNative28Matrix` blocked in drive status waits; the full failure output is retained in `verification-20261026T1503Z/RESULT.txt` with its checksum.
- No live cloud/account/auth/credential operation was requested or performed. No hosted CI, PR, merge, or deployment action has been taken; the candidate remains uncommitted.
- Additional bounded qualification at 2026-10-26T15:32:01Z passed: `git diff --check`, `gofmt` cleanliness, focused `go test -race ./internal/cloudsigmaapi ./internal/provider -run 'TestOfflineRemoteSnapshotNativeCreate|^$' -count=1`, `go vet ./internal/cloudsigmaapi ./internal/provider`, and `go build ./...`. Receipt: `verification-20261026T1529Z/RESULT.txt`; checksum: `1fe75926d26e772193184775fcf1f560c77eede4b544e697b871c8db087e6356`.
- The repository-wide timeout receipt remains preserved and unchanged; no failure was overwritten or relabeled. No matching canonical PRD was returned by the current PRD search, so this report remains the durable candidate handoff.

## Files
 M internal/provider/provider.go
?? internal/cloudsigmaapi/remote_snapshots.go
?? internal/provider/offline_remote_snapshot_test.go
?? internal/provider/resource_cloudsigma_remote_snapshot.go
?? terraform-remote-snapshot-native-20261026/
