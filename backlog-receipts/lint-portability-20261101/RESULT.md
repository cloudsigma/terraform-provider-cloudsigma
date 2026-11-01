# Lint portability remediation — 2026-11-01

The offline isolation guard is Linux amd64 specific because it uses seccomp TSYNC and Linux syscall numbers. The generated guard template now has a `//go:build linux && amd64` constraint. `scripts/offline.py` selects that enforced template only for Linux amd64 and uses a portable canary template elsewhere, which skips the Linux-only isolation test rather than weakening Linux CI.

`.golangci.yaml` excludes generated `build/offline` artifacts from lint package loading. This prevents stale/generated overlay files from being interpreted as a mixed package while preserving lint coverage for repository sources.

Verification:

- `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOPROXY=off go test -vet=off -overlay ... -c ... ./internal/cloudsigmaapi` — PASS; ELF amd64 test binary compiled with the seccomp guard.
- `GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 GOPROXY=off` with the portable guard overlay — PASS; Mach-O arm64 test binary compiled.
- Darwin compilation with the Linux guard overlay fails on the unsupported syscall symbols as expected, proving the build constraint must be selected per target.
- `golangci-lint run` with generated Linux guard artifacts present and `build/offline` excluded — PASS, 0 issues.
- `python3 -m py_compile scripts/offline.py` and `git diff --check` — PASS.

No dependency download or installation was performed.
