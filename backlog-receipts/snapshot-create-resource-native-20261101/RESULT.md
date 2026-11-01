# Native snapshot create resource routing — 2026-11-01

The Terraform snapshot resource now routes Basic-auth create through the
provider-owned native `CreateSnapshot` transport. Bearer create remains on the
legacy SDK path until bearer snapshot mutation parity is separately qualified.
The broader Native28/Native29 matrix fixtures now explicitly qualify Basic native
POST status/routing while preserving the bearer and legacy paths.
The existing native waiter and caller-context final read remain in use, and the
legacy path is unchanged. Native create failures require an acknowledged
snapshot UUID before state is written.

## Qualification

- Native API transport remains minimal: `drive.uuid` and `name` only, no
  response-only fields or full drive object.
- HTTP 201 envelope identity, waiter status, final read projection, malformed
  responses, invalid identity, missing drive, 404 waiter, and final read error
  behavior are covered by synthetic fixtures.
- Existing state/import and local snapshot semantics are unchanged.

## Validation

```text
go test ./internal/cloudsigmaapi -run 'Test(CreateSnapshot|UpdateSnapshot|SnapshotWrite)' -count=1   PASS
go test ./internal/provider -run 'TestOfflineNativeSnapshot|TestOfflineNativeSnapshotDelete' -count=1   PASS
go vet ./internal/cloudsigmaapi ./internal/provider                                                      PASS
go build ./...                                                                                             PASS
git diff --check                                                                                           PASS
```

No live API, account, or billable operation was used.
