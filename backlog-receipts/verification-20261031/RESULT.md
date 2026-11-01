# Verification receipt — 2026-10-31

## Passing gates

- `GOPROXY=off go test ./internal/cloudsigmaapi -count=1 -timeout=240s` — exit 0
- `GOPROXY=off go test -race ./internal/cloudsigmaapi -count=1 -timeout=300s` — exit 0
- Targeted offline provider tests and race tests covering native configure/state,
  schemas, data-source coverage, import protocol, RemoteSnapshot draft,
  native Tag, and native License — exit 0
- `GOPROXY=off go vet ./...` — exit 0
- `GOPROXY=off go build ./...` — exit 0
- `git diff --check` — exit 0
- `GOPROXY=off go test ./internal/provider -run '^TestOffline' -count=1 -timeout=240s` — exit 0
- Matching `-race` command — exit 0
- `make lint` using the cached local tool — exit 0 (`0 issues`)
- `GOPROXY=off go test ./internal/provider -run '^TestOfflineTagResourceImportRead$' -count=1 -timeout=120s` — exit 0
- Matching import-refresh `-race` command — exit 0
- `GOPROXY=off go test ./internal/provider -run 'TestOfflineCapabilitiesDataSource|TestOfflineRegisteredSchemas|TestOfflineDatasourceCoverage|TestOfflineNativeConfigure' -count=1 -timeout=180s` — exit 0
- Full `TestOffline` provider suite and matching race command after capabilities data-source registration — exit 0
- `GOPROXY=off go test ./internal/cloudsigmaapi -run '^TestJobs' -count=1 -timeout=120s` — exit 0
- QA read-only native checks against `next` (profile, capabilities, locations,
  Jobs) — exit 0; receipt in `live-qa-readonly-next-20261031`.
- `GOPROXY=off go test ./internal/cloudsigmaapi -run 'TestDriveScheduler|TestCreateBackupScheduler|TestCreateRemoteSnapshot' -count=1 -timeout=120s` — exit 0
- Matching cloud API race command — exit 0
- `TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform GOPROXY=off go test ./internal/provider -run '^(TestProvider|TestOffline)' -count=1 -timeout=5m` — exit 0
- Matching provider/unit race command — exit 0
- `TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform GOPROXY=off go test ./internal/provider -run '^TestProviderConfigure_invalidCredentials$' -count=1 -timeout=120s -v` — exit 0

## Preserved blockers

- Broad `GOPROXY=off go test ./internal/provider -count=1 -timeout=180s` was
  run without an acceptance filter and timed out after 3 minutes in the
  existing native/acceptance matrix (`TestNative28Matrix` and acceptance tests).
  The exact panic/stack output remains in the command result for this turn; no
  pass is claimed for that broad command.
- The proposed Terraform CLI stub was removed after independent review. The
  signing-key issue is now resolved by the verified Terraform 1.17.1 install;
  Terraform-dependent provider/unit tests pass with the real CLI.
- The lint-tool issue was fixed independently by building the cached
  golangci-lint source offline into the ignored `tools/bin` path; no dependency
  download was performed.
- Broad `GOPROXY=off go test ./internal/provider -count=1 -timeout=300s` now
  passes the Terraform-helper setup but still times out in the existing
  `TestNative28Matrix` while exercising long waiter/error phases. No pass is
  claimed for that broad command; targeted offline and native API suites pass.
- The real-CLI full provider run with a 10-minute timeout reproduced the same
  existing `TestNative28SnapshotLifecycle` waiter/error timeout; no
  full-package pass is claimed.
