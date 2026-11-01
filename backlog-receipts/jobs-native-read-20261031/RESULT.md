# Native Jobs read qualification — 2026-10-31

Added provider-owned read methods for the documented asynchronous Jobs API:

- `Client.Jobs` reads `/api/2.0/jobs/?limit=0` and preserves recursive child
  jobs and dynamic `data` fields.
- `Client.Job` reads `/api/2.0/jobs/{uuid}/` by validated UUID.
- No polling, replay, completion inference, or Terraform lifecycle resource was
  added.

Synthetic tests cover Basic authentication, collection/detail routes, nested
children, dynamic progress data, malformed collections, and invalid IDs.

Validation:

- `GOPROXY=off go test ./internal/cloudsigmaapi -run '^TestJobs' -count=1 -timeout=120s` — PASS
