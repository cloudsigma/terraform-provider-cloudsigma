---
page_title: "cloudsigma_scheduler Resource - cloudsigma"
subcategory: ""
description: |-
  Manages a CloudSigma backup scheduler using the pinned native scheduler contract. `definition_json` must be the complete writable scheduler projection.
---

# cloudsigma_scheduler (Resource)

Manages a CloudSigma backup scheduler using the pinned native scheduler contract. `definition_json` must be the complete writable scheduler projection.

## Schema

### Optional

- `definition_json` (String) Complete writable scheduler JSON projection.

### Read-Only

- `id` (String) The scheduler UUID.
- `json` (String) The last acknowledged scheduler document.
