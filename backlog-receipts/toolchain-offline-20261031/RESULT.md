# Terraform and test-tool remediation — 2026-10-31

**Correction after independent review:** the proposed executable stub was removed.
It could give a misleading success for tests that actually require Terraform.
The signing-key issue is resolved by installing a verified Terraform CLI.

## Terraform CLI installation

- macOS Command Line Tools updated via `softwareupdate` to Xcode 27.2.
- Official HashiCorp Terraform 1.17.1 Darwin arm64 archive checksum verified:
  `b1ecb3fe0d0a0290c1ab6f83d967d43f25da05d0b671ee0fb355252268dce95a`.
- Homebrew’s official `hashicorp/tap/terraform` formula installed successfully
  at `/opt/homebrew/bin/terraform`.
- `terraform version` reports Terraform v1.17.1 on darwin_arm64.

## Terraform helper signing key

The repository uses terraform-plugin-testing v1.9.0, whose helper initializes
Terraform before checking whether `TF_ACC` is set. With no Terraform binary,
offline runs attempted hc-install and failed on an expired OpenPGP signing key.

The attempted test-only workaround did not fix the underlying signing-key
issue and was removed. Terraform-driven tests and live acceptance require a
real verified CLI via `TF_ACC_TERRAFORM_PATH` or a working verified installer.

## Lint tool

The cached golangci-lint source was built offline into the ignored
`tools/bin/golangci-lint` path. No dependency was downloaded or installed.

## Verification

- `TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform GOPROXY=off go test ./internal/provider -run '^(TestProvider|TestOffline)' -count=1 -timeout=5m` — PASS
- Matching race command — PASS
- `TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform GOPROXY=off go test ./internal/provider -run '^TestProviderConfigure_invalidCredentials$' -count=1 -timeout=120s -v` — PASS
- `make lint` — PASS (`0 issues`)

## Remaining broad-suite evidence

- Full provider tests with the real CLI still timed out after 10 minutes in the
  existing `TestNative28SnapshotLifecycle` waiter/error matrix.
- The targeted Native28 race run failed after 528 seconds with an existing data
  race in `native28_matrix_test.go` around shared fixture state.
