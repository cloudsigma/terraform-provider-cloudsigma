---
page_title: "CloudSigma: cloudsigma_snapshot"
---

# cloudsigma_snapshot

Provides a CloudSigma snapshot resource.


## Example Usage

```hcl
resource "cloudsigma_snapshot" "snapshot" {
  drive  = "my_policy"
  location = "ZRH"
  name = "my snapshot"
}
```


## Argument Reference

The following arguments are supported:

* `drive` - (Required) The UUID of the drive
* `name` - (Required) The name of the snapshot


## Attributes Reference

In addition to all above arguments, the following attributes are exported:

* `resource_uri`
* `status`
* `timestamp`
