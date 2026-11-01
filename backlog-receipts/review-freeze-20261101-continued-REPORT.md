# Review freeze — 2026-11-01 continuation

HEAD: 160845a69d5481394d15ee6ecbbece8007ca3da4
Branch: main
Remote: https://github.com/cloudsigma/terraform-provider-cloudsigma.git

## Working tree
 M internal/cloudsigmaapi/client.go
 M internal/cloudsigmaapi/drive_attachment.go
 M internal/cloudsigmaapi/drive_clone.go
 M internal/cloudsigmaapi/drive_create.go
 M internal/cloudsigmaapi/drive_delete.go
 M internal/cloudsigmaapi/drives.go
 M internal/cloudsigmaapi/keypair_create.go
 M internal/cloudsigmaapi/keypair_create_test.go
 M internal/cloudsigmaapi/keypair_delete.go
 M internal/cloudsigmaapi/keypair_update.go
 M internal/cloudsigmaapi/library_drives_test.go
 M internal/cloudsigmaapi/server_attachment.go
 M internal/cloudsigmaapi/server_create.go
 M internal/cloudsigmaapi/server_delete.go
 M internal/cloudsigmaapi/server_power.go
 M internal/cloudsigmaapi/snapshot_delete.go
 M internal/cloudsigmaapi/snapshots.go
 M internal/cloudsigmaapi/tags.go
 M internal/provider/data_source_cloudsigma_license.go
 M internal/provider/data_source_cloudsigma_tag.go
 M internal/provider/native28_matrix_test.go
 M internal/provider/native28_snapshot_lifecycle_test.go
 M internal/provider/offline_datasource_coverage_test.go
 M internal/provider/offline_native_library_drive_test.go
 M internal/provider/offline_native_license_test.go
 M internal/provider/offline_native_ssh_key_delete_test.go
 M internal/provider/offline_native_ssh_key_test.go
 M internal/provider/offline_native_ssh_key_update_test.go
 M internal/provider/offline_native_tag_test.go
 M internal/provider/offline_native_test.go
 M internal/provider/offline_reads_test.go
 M internal/provider/offline_schema_test.go
 M internal/provider/provider.go
 M internal/provider/resource_cloudsigma_snapshot.go
 M internal/provider/resource_cloudsigma_ssh_key.go
 M internal/provider/resource_cloudsigma_tag.go
?? backlog-receipts/api-docs-gap-review-20261031/
?? backlog-receipts/api-probe-matrix-20261031/
?? backlog-receipts/capabilities-native-read-20261031/
?? backlog-receipts/current-usage-native-read-20261031/
?? backlog-receipts/import-31-20261029/
?? backlog-receipts/import-31-native-tag-refresh-20261031/
?? backlog-receipts/jobs-datasource-native-20261031/
?? backlog-receipts/jobs-native-read-20261031/
?? backlog-receipts/live-qa-readonly-next-20261031/
?? backlog-receipts/native-bearer-write-parity-20261031/
?? backlog-receipts/native-license-token-20261029/
?? backlog-receipts/native-remote-snapshot-bearer-20261031/
?? backlog-receipts/native-ssh-bearer-parity-20261031/
?? backlog-receipts/native-ssh-key-token-parity-20261031/
?? backlog-receipts/native-ssh-resource-sdk-removal-20261101/
?? backlog-receipts/native-tag-crud-20261029/
?? backlog-receipts/native28-test-remediation-20261031/
?? backlog-receipts/notification-acl-native-reads-20261101/
?? backlog-receipts/pricing-discount-native-reads-20261031/
?? backlog-receipts/review-freeze-20261031/
?? backlog-receipts/review-freeze-20261101-continued-REPORT.md
?? backlog-receipts/scheduler-30-blocker-20261029/
?? backlog-receipts/scheduler-assignment-native-20261031/
?? backlog-receipts/scheduler-create-native-20261029/
?? backlog-receipts/scheduler-create-native-20261031-addendum.md
?? backlog-receipts/scheduler-empirical-next-20261031/
?? backlog-receipts/scheduler-read-datasource-native-20261031/
?? backlog-receipts/snapshot-native-waiter-20261101/
?? backlog-receipts/snapshot-native-write-20261031/
?? backlog-receipts/toolchain-offline-20261031/
?? backlog-receipts/turlo-access-20261101/
?? backlog-receipts/usage-native-reads-20261031/
?? backlog-receipts/verification-20261031/
?? backlog-receipts/virtual-router-native-read-20261031/
?? docs/data-sources/acls.md
?? docs/data-sources/backup_schedulers.md
?? docs/data-sources/balance.md
?? docs/data-sources/burstusage.md
?? docs/data-sources/capabilities.md
?? docs/data-sources/current_usage.md
?? docs/data-sources/dailyburstusage.md
?? docs/data-sources/discount.md
?? docs/data-sources/jobs.md
?? docs/data-sources/notification_contacts.md
?? docs/data-sources/notification_preferences.md
?? docs/data-sources/pricing.md
?? docs/data-sources/usage.md
?? docs/data-sources/virtual_routers.md
?? internal/cloudsigmaapi/backup_schedulers.go
?? internal/cloudsigmaapi/backup_schedulers_test.go
?? internal/cloudsigmaapi/capabilities.go
?? internal/cloudsigmaapi/capabilities_test.go
?? internal/cloudsigmaapi/current_usage.go
?? internal/cloudsigmaapi/current_usage_test.go
?? internal/cloudsigmaapi/drive_scheduler.go
?? internal/cloudsigmaapi/drive_scheduler_test.go
?? internal/cloudsigmaapi/jobs.go
?? internal/cloudsigmaapi/jobs_test.go
?? internal/cloudsigmaapi/notifications_acls.go
?? internal/cloudsigmaapi/notifications_acls_test.go
?? internal/cloudsigmaapi/remote_snapshots.go
?? internal/cloudsigmaapi/remote_snapshots_test.go
?? internal/cloudsigmaapi/snapshot_wait.go
?? internal/cloudsigmaapi/snapshot_wait_test.go
?? internal/cloudsigmaapi/snapshot_write_test.go
?? internal/cloudsigmaapi/tags_native_crud_test.go
?? internal/cloudsigmaapi/usage.go
?? internal/cloudsigmaapi/usage_test.go
?? internal/cloudsigmaapi/virtual_routers.go
?? internal/cloudsigmaapi/virtual_routers_test.go
?? internal/provider/data_source_cloudsigma_backup_schedulers.go
?? internal/provider/data_source_cloudsigma_capabilities.go
?? internal/provider/data_source_cloudsigma_current_usage.go
?? internal/provider/data_source_cloudsigma_jobs.go
?? internal/provider/data_source_cloudsigma_notifications_acls.go
?? internal/provider/data_source_cloudsigma_usage.go
?? internal/provider/data_source_cloudsigma_virtual_routers.go
?? internal/provider/offline_backup_schedulers_test.go
?? internal/provider/offline_capabilities_test.go
?? internal/provider/offline_current_usage_test.go
?? internal/provider/offline_jobs_test.go
?? internal/provider/offline_notification_acl_test.go
?? internal/provider/offline_remote_snapshot_test.go
?? internal/provider/offline_tag_resource_test.go
?? internal/provider/offline_usage_datasource_test.go
?? internal/provider/offline_virtual_router_test.go
?? internal/provider/resource_cloudsigma_remote_snapshot.go
?? terraform-remote-snapshot-native-20261026/

## Tracked diff stat
 internal/cloudsigmaapi/client.go                   |  45 +++++-
 internal/cloudsigmaapi/drive_attachment.go         |   2 +-
 internal/cloudsigmaapi/drive_clone.go              |   2 +-
 internal/cloudsigmaapi/drive_create.go             |   2 +-
 internal/cloudsigmaapi/drive_delete.go             |   2 +-
 internal/cloudsigmaapi/drives.go                   |   1 +
 internal/cloudsigmaapi/keypair_create.go           |   2 +-
 internal/cloudsigmaapi/keypair_create_test.go      |  22 +++
 internal/cloudsigmaapi/keypair_delete.go           |   2 +-
 internal/cloudsigmaapi/keypair_update.go           |   2 +-
 internal/cloudsigmaapi/library_drives_test.go      |   5 +-
 internal/cloudsigmaapi/server_attachment.go        |   2 +-
 internal/cloudsigmaapi/server_create.go            |   2 +-
 internal/cloudsigmaapi/server_delete.go            |   2 +-
 internal/cloudsigmaapi/server_power.go             |   2 +-
 internal/cloudsigmaapi/snapshot_delete.go          |   2 +-
 internal/cloudsigmaapi/snapshots.go                | 171 +++++++++++++++++++++
 internal/cloudsigmaapi/tags.go                     | 161 +++++++++++++++++++
 .../provider/data_source_cloudsigma_license.go     |  15 +-
 internal/provider/data_source_cloudsigma_tag.go    |   9 +-
 internal/provider/native28_matrix_test.go          |  61 +++++---
 .../provider/native28_snapshot_lifecycle_test.go   |   7 +-
 .../provider/offline_datasource_coverage_test.go   |   2 +-
 .../provider/offline_native_library_drive_test.go  |   5 +
 internal/provider/offline_native_license_test.go   |  24 +--
 .../provider/offline_native_ssh_key_delete_test.go |   7 +-
 internal/provider/offline_native_ssh_key_test.go   |  16 +-
 .../provider/offline_native_ssh_key_update_test.go |   7 +-
 internal/provider/offline_native_tag_test.go       |   9 +-
 internal/provider/offline_native_test.go           |   6 +-
 internal/provider/offline_reads_test.go            |  20 +--
 internal/provider/offline_schema_test.go           |   2 +-
 internal/provider/provider.go                      |  14 ++
 internal/provider/resource_cloudsigma_snapshot.go  |   6 +-
 internal/provider/resource_cloudsigma_ssh_key.go   | 126 ++++-----------
 internal/provider/resource_cloudsigma_tag.go       |  38 ++---
 36 files changed, 569 insertions(+), 234 deletions(-)

## New native slices
internal/cloudsigmaapi/snapshot_wait.go
internal/cloudsigmaapi/snapshot_wait_test.go
internal/cloudsigmaapi/notifications_acls.go
internal/cloudsigmaapi/notifications_acls_test.go
internal/provider/data_source_cloudsigma_notifications_acls.go
internal/provider/offline_notification_acl_test.go
internal/provider/resource_cloudsigma_ssh_key.go

## Receipts
backlog-receipts/turlo-access-20261101/RESULT.md
backlog-receipts/notification-acl-native-reads-20261101/RESULT.md
backlog-receipts/snapshot-native-waiter-20261101/RESULT.md
backlog-receipts/native-ssh-resource-sdk-removal-20261101/RESULT.md

## Validation
native API full tests PASS
provider offline normal/race PASS
focused snapshot waiter normal/race PASS
go vet ./... PASS
go build ./... PASS
make lint PASS
make docs PASS
git diff --check PASS
