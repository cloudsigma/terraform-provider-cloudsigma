# Terraform provider import/read matrix — 2026-10-29

Issue #31 requested support for `terraform import` across provider resources.
The current provider exposes five managed resource types. Each has an
identifier passthrough importer, and the offline protocol test exercises every
one through the actual Terraform v6 mux.

| Resource | Implementation | Import callback | Imported state | Refresh after import | Missing-object behavior | No-op qualification |
|---|---|---|---|---|---|---|
| `cloudsigma_drive` | SDKv2 legacy resource | `ImportStatePassthroughContext` | ID only | `resourceCloudSigmaDriveRead` | API/read path decides; existing legacy behavior retained | Protocol import only; live no-op requires account fixture |
| `cloudsigma_server` | SDKv2 legacy resource | `ImportStatePassthroughContext` | ID only | `resourceCloudSigmaServerRead` | API/read path decides; existing legacy behavior retained | Protocol import only; live no-op requires account fixture |
| `cloudsigma_snapshot` | Framework resource | `ImportStatePassthroughID` | ID only | Framework `Read`; native read available, create/update still SDK-backed | 404 removes state; other errors preserve state | Offline identifier import; refresh/no-op requires API fixture |
| `cloudsigma_ssh_key` | Framework resource | `ImportStatePassthroughID` | ID only | Framework `Read`; native Basic read, SDK token fallback remains | 404 removes state; other errors preserve state | Offline identifier import; refresh/no-op requires API fixture |
| `cloudsigma_tag` | Framework resource | `ImportStatePassthroughID` | ID only | Framework `Read`; provider-owned native GET | 404 removes state | Offline identifier import plus native CRUD/read fixtures; no live no-op claim |

`cloudsigma_remote_snapshot` is not registered because its current draft lacks
qualified Read/Delete/import lifecycle semantics. It is excluded from the
supported import matrix until the public API supplies those contracts.

The protocol test intentionally verifies identifier-only state and does not
pretend that an import succeeded against the cloud or that Terraform produced
a no-op plan. A full no-op matrix needs disposable API fixtures or separately
authorized acceptance runs per resource, plus drift checks for each computed
field. Those are distinct from the import callback itself.

## Verification

`GOPROXY=off go test ./internal/provider -run '^TestOfflineImportProtocol$' -count=1`
passes against the mux and both provider implementations. The exact test also
covers repeated IDs and an empty ID for Framework resources, ensuring state is
not reused between imports. Existing merged receipt:
`backlog-receipts/import-31-20261027/RESULT.txt`.

## Disposition

The provider-wide import callback portion of #31 is implemented and covered.
The remaining no-op/drift qualification is an acceptance matrix requiring
resource fixtures and does not justify claiming complete import verification.
