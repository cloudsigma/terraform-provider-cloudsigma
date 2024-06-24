---
page_title: "CloudSigma: cloudsigma_drive"
---

# Resource: cloudsigma_drive

Provides a CloudSigma Drive resource which can be attached to a Server.


## Examples

### Basic

```hcl
resource "cloudsigma_drive" "foobar" {
  media = "disk"
  name  = "foobar"
  size  = 5 * 1024 * 1024 * 1024 # 5GB
}
```

### With storage type

```hcl
resource "cloudsigma_drive" "foobar" {
  media = "disk"
  name  = "foobar"
  size  = 5 * 1024 * 1024 * 1024 # 5GB
  storage_type = "dssd"
}
```

### With additional tags
```hcl
resource "cloudsigma_drive" "foobar" {
  media = "disk"
  name  = "foobar"
  size  = 5 * 1024 * 1024 * 1024 # 5GB

  tags = [
    "first-tag-uuid",
    "second-tag-uuid",
  ]
}
```


## Argument Reference

The following arguments are supported:

* `clone_drive_id` - (Optional) UUID of the drive that will be cloned
* `media` - (Required) Media representation type. It can be `cdrom` or `disk`
* `name` - (Required) Human readable name of the drive
* `size` - (Required) Size of the drive in bytes
* `storage_type` - (Optional) Drive storage type, cannot be changed after drive creation
* `tags` - (Optional) A list of the tags UUIDs to be applied to the drive.


## Attributes Reference

The following attributes are exported:

* `media` - (Required) Media representation type. It can be `cdrom` or `disk`
* `name` - (Required) Human readable name of the drive
* `size` - (Required) Size of the drive in bytes
