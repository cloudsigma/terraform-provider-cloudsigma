# QA read-only native transport check — 2026-10-31

## Account and scope

- Credential source: 1Password item `Next QA account` in vault `Clawdbot`.
- Location: `next` (`https://next.cloudsigma.com`).
- Scope: read-only native transport checks only. No resource creation,
  mutation, deletion, Terraform acceptance, or budget charge was performed.
- Secret values were used in process only and were not written to files, logs,
  chat, or repository state. The clipboard was cleared after the check.

## Checks

The native client authenticated successfully and returned valid responses for:

- `GET /api/2.0/profile/`
- `GET /api/2.0/capabilities/?limit=0`
- `GET /api/2.0/locations/`
- `GET /api/2.0/jobs/?limit=0`

Command:

`CLOUDSIGMA_QA_USERNAME=<in-memory> CLOUDSIGMA_QA_PASSWORD=<in-memory> GOPROXY=off go test ./internal/cloudsigmaapi -run '^TestLiveQANativeReadOnly$' -count=1 -timeout=120s`

Result: PASS, exit 0.

The temporary test was removed after execution.
