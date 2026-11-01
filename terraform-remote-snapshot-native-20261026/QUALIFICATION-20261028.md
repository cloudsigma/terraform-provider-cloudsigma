# Native RemoteSnapshots qualification update

Date: 2026-10-28

## PR state

The connected GitHub repository has no open pull requests. The recent native
transport PRs are merged, and Dependabot PR #125 is closed and superseded by
merged PR #126. No public write was performed for this update.

## Delivery boundary

The preserved draft managed resource is not registered by the provider. Its
empty `Read`, unsupported `Update`/`Delete`, and passthrough import cannot be
presented as a Terraform lifecycle before the API documents those semantics.
The original registration patch remains preserved at
`outputs/dirty-main.patch` and in the prior report; the draft source remains in
the working tree for a later qualified extension.

The delivered native slice is the provider-owned POST transport and projection:

- accepts a UUID string or a minimal `{ "uuid": "..." }` drive reference;
- emits only `drive` and optional `location` writable fields;
- supports the API's bare-object and `objects` batch request projection;
- requires HTTP 201 and an ordered `objects` response matching the request;
- preserves returned UUID, location, and status fields without polling;
- ignores response-only `drive_meta` and rejects complete drive response objects;
- refuses redirects and unexpected statuses, and classifies malformed,
  transport, cancellation, and deadline outcomes without response-body leakage.

Location remains syntactically bounded only. Configured writable destination
alias resolution is still a documented API/provider contract blocker, as are
GET-by-UUID, DELETE, missing-object, and import semantics. No full Terraform
CRUD resource is claimed.

## Verification

See `verification-20261028T*.txt` for exact commands and results. Offline
provider tests, focused native tests, race tests, vet, build, formatting, and
diff checks pass with `GOPROXY=off`. `make lint` is blocked because the
repository-local `tools/bin/golangci-lint` binary is absent; no installation or
download was attempted. Prior full-suite timeout evidence remains unchanged.
