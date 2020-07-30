---
page_title: "CloudSigma: cloudsigma_drive_attachment

---

# Resource: cloudsigma_drive_attachment

Manages attaching a Drive to a Server.


## Example Usage

```hcl
resource "cloudsigma_drive" "foobar" {
  media = "disk"
  name  = "foobar"
  size  = 5 * 1024 * 1024 * 1024 # 5GB
}

resource "cloudsigma_server" "foobar" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "foobar"
  vnc_password = "cloudsigma"
}

resource "cloudsigma_drive_attachment" "foobar" {
  drive_id   = cloudsigma_drive.foobar.id
  server_id  = cloudsigma_server.foobar.id
}
```


## Argument Reference

The following arguments are supported:

* `drive_id` - (Required) UUID of the Drive to be attached to the Server
* `server_id` - (Required) UUID of the Server to attach the Volume to


## Attributes Reference

The following attributes are exported:

* `id` - The unique identifier for the drive attachment
