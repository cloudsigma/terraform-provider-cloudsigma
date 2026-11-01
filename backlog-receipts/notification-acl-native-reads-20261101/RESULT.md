# Native notification and ACL reads — 2026-11-01

Implemented read-only provider-owned collection coverage:

- `GET /api/2.0/notification_contacts/?limit=0`
- `GET /api/2.0/notification_preferences/?limit=0`
- `GET /api/2.0/acls/?limit=0`

Responses are preserved as dynamic JSON, including pagination metadata and deployment-specific fields. Registered data sources:

- `cloudsigma_notification_contacts`
- `cloudsigma_notification_preferences`
- `cloudsigma_acls`

Basic and bearer authentication, strict status/decode/trailing-response handling, redirect refusal, offline tests, and race tests are covered. No notification mutation, ACL mutation, ownership, or Terraform lifecycle semantics were inferred. Public docs show mutation endpoints, but notification preference response status examples vary and ACL ownership/sharing rules require a separate lifecycle contract.

Validation:

- Focused native/provider normal and race tests — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `make lint` — passed with 0 issues.
- `make docs` — passed; generated data-source docs.
- `git diff --check` — passed.
