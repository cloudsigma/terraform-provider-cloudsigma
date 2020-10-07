---
page_title: "CloudSigma: cloudsigma_remote_snapshot"

---

# cloudsigma_remote_snapshot

Provides a CloudSigma remote snapshot resource.


## Example Usage

```hcl
resource "cloudsigma_remote_snapshot" "my_snapshot" {
  drive  = "my_policy"
  location = "ZRH"
  name = "my snapshot"
}
```


## Argument Reference

The following arguments are supported:

* `drive` - (Required) The UUID of the drive
* `location` - (Required) The location of the remote snapshot
* `name` - (Required) The name of the remote snapshot


## Attributes Reference

In addition to all above arguments, the following attributes are exported:

* `resource_uri`
* `status`
