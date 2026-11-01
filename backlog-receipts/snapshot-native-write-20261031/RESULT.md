# Native snapshot write transport — 2026-10-31

## Scope

Added provider-owned HTTP transport for snapshot create and update in
`internal/cloudsigmaapi/snapshots.go` without changing the registered
Terraform resource routing. The transport intentionally remains a library
slice until resource-level waiter/state compatibility is reviewed.

## Contract enforced

- Create: `POST /api/2.0/snapshots/`, minimal `objects` envelope containing
  only `drive.uuid` and `name`, strict HTTP `201`, exactly one acknowledged
  UUID.
- Update: `PUT /api/2.0/snapshots/{uuid}/`, minimal `drive.uuid` and `name`,
  strict HTTP `200`, response UUID must match the URL UUID.
- Basic and bearer authentication use the provider-owned `authenticate`
  boundary; redirects are not followed by the client policy.
- Response-only fields are decoded into state projections but never emitted
  into mutation payloads. No readiness polling or retry is performed.
- Unexpected statuses, malformed/trailing JSON, invalid identity, transport,
  cancellation, deadline, unsupported encoding, and oversized responses are
  classified without response-body leakage.

## Verification

```text
GOPROXY=off GOSUMDB=off go test ./internal/cloudsigmaapi -run 'Test(CreateSnapshot|UpdateSnapshot|SnapshotWrite)' -count=1
ok   github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi  0.262s
```

The existing Terraform snapshot resource still routes create/update through
the legacy SDK. A separate migration must qualify waiter behavior, identity
and state parity, and existing-state/import compatibility before switching
that resource.

The transport now wraps every write in the provider's 60-second request
deadline before cloning the HTTP client timeout. A focused deadline assertion
and the race variant both pass:

```text
go test ./internal/cloudsigmaapi -run 'SnapshotWrite|CreateSnapshot|UpdateSnapshot' -count=1
go test -race ./internal/cloudsigmaapi -run 'SnapshotWrite|CreateSnapshot|UpdateSnapshot' -count=1
```
