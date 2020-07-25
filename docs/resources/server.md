---
page_title: "CloudSigma: cloudsigma_server

---

# Resource: cloudsigma_server

Provides a CloudSigma Server resource. This can be used to create, modify,
and delete Servers.


## Example Usage

```hcl
# Create a new server with name 'web'
resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "cloudsigma"
}
```


## Argument Reference

The following arguments are supported:

* `cpu` - (Required) Server's CPU Clock speed measured in MHz
* `memory` - (Required) Server's RAM measured in bytes
* `name` - (Required) Human readable name of server
* `vnc_password` - (Required) VNC Password to connect to server


## Attributes Reference

The following attributes are exported:

* `cpu` - Server's CPU Clock speed measured in MHz
* `memory` - Server's RAM measured in bytes
* `name` - Human readable name of server
* `vnc_password` - VNC Password to connect to server
