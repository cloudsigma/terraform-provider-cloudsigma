---
page_title: "CloudSigma: cloudsigma_drive

---

# Resource: cloudsigma_drive

Provides a CloudSigma Drive resource which can be attached to a Server.


## Example Usage

```hcl
# Create a new drive with name 'foobar'
resource "cloudsigma_drive" "foobar" {
  media = "disk"
  name  = "foobar"
  size  = 5 * 1024 * 1024 * 1024 # 5GB
}
```


## Argument Reference

The following arguments are supported:

* `clone_drive_id` - (Optional) UUID of the drive that will be cloned
* `media` - (Required) Media representation type. It can be `cdrom` or `disk`
* `name` - (Required) Human readable name of the drive
* `size` - (Required) Size of the drive in bytes


## Attributes Reference

The following attributes are exported:

* `media` - (Required) Media representation type. It can be `cdrom` or `disk`
* `name` - (Required) Human readable name of the drive
* `size` - (Required) Size of the drive in bytes
