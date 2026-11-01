# Native Basic snapshot update resource routing — 2026-11-02

The Terraform snapshot resource now routes Basic-auth update through the
provider-owned native `UpdateSnapshot` transport. Bearer update remains on the
legacy SDK path until bearer mutation parity is separately qualified.

## Contract and state behavior

- `PUT /api/2.0/snapshots/{uuid}/` emits only `drive.uuid` and `name`.
- Native transport requires HTTP 200 and matching response UUID; response-only
  fields such as `drive_meta` are decoded for state only and never sent.
- Native Basic update preserves the existing resource state projection and
  reports API failures without replacing prior state.
- Legacy local snapshot update and bearer routing remain unchanged.

## Validation

```text
go test ./internal/provider -run 'TestOfflineNativeSnapshotUpdate|TestOfflineNativeSnapshotUpdateErrorPreservesState' -count=1   PASS
go test -race ./internal/provider -run 'TestOfflineNativeSnapshotUpdate|TestOfflineNativeSnapshotUpdateErrorPreservesState' -count=1 PASS
go test ./internal/provider -run '<exact offline provider suite selector>' -count=1                                      PASS (465 selected)
go vet ./internal/provider                                                                                       PASS
go build ./...                                                                                                  PASS
git diff --check                                                                                                PASS
```

No live API, account, or billable operation was used.
