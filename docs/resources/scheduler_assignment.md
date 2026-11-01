---
page_title: "cloudsigma_scheduler_assignment Resource - cloudsigma"
subcategory: ""
description: |-
  Owns only the drive-to-scheduler association.
---

# cloudsigma_scheduler_assignment (Resource)

Owns only the drive-to-scheduler association. Scheduler and drive resources do not write this association.

## Schema

### Required

- `drive_uuid` (String)
- `scheduler_uuid` (String)

### Read-Only

- `id` (String) The drive UUID.
