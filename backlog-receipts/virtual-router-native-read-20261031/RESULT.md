# Native virtual-router read — 2026-10-31

Implemented provider-owned read-only virtual-router collection coverage:

- `GET /api/2.0/virtual_routers/?limit=0` using the current production route.
- Dynamic JSON preservation for subscription- and deployment-specific router configuration.
- Registered `cloudsigma_virtual_routers` data source with stable collection identity.
- Basic and bearer authentication, strict status/decode/trailing handling, and no lifecycle/action semantics.

The docs describe complete configuration writes, separate lifecycle/configuration status, and subscription gates. No managed Terraform resource or action was added.

Validation:

- Native normal and race virtual-router tests — passed.
- Provider offline normal and race tests — passed.
- Full `internal/cloudsigmaapi` tests — passed.
- Full provider offline normal and race suites — passed.
- `go vet ./...`, `go build ./...`, `make lint`, `make docs`, and `git diff --check` — passed.
- OPTIONS against `next.cloudsigma.com` advertised `GET,PUT,DELETE,POST`; this is capability evidence only and no live mutation was performed.
