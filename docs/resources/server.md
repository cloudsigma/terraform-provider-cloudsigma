---
page_title: "CloudSigma: cloudsigma_server

---

# Resource: cloudsigma_server

Provides a CloudSigma Server resource. This can be used to create, modify,
and delete Servers.


## Examples

### Basic

```hcl
resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "cloudsigma"
}
```

### With additional drives

```hcl
resource "cloudsigma_drive" "data" {
  media = "disk"
  name  = "web-data"
  size  = 5 * 1024 * 1024 * 1024 # 5GB
}

resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "cloudsigma"

  drive {
    uuid = cloudsigma_drive.data.id
  }
}
```

### With static IP address

```hcl
data "cloudsigma_ip" "load_balancer" {
  filter {
    name   = "uuid"
    values = ["31.171.251.11"]
  }
}

resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "cloudsigma"

  network {
    ipv4_address = data.cloudsigma_ip.load_balancer.id
    type = "static"
  }
}
```


## Argument Reference

The following arguments are supported:

* `cpu` - (Required) Server's CPU Clock speed measured in MHz
* `memory` - (Required) Server's RAM measured in bytes
* `name` - (Required) Human readable name of server
* `vnc_password` - (Required) VNC Password to connect to server
* `drive` - (Optional) Drive attached to the server on creation
    - uuid - (Required) The UUID of the drive
* `network` - (Optional) Network interface card attached to the server
    - ipv4_address - (Optional) The IP address reference. Only used with `static` type
    - type - (Optional) Configuration type. Valid values: `dhcp`, `static`, `manual`
    - vlan_uuid - (Optional) The UUID of the VLAN reference


## Attributes Reference

The following attributes are exported:

* `cpu` - Server's CPU Clock speed measured in MHz
* `memory` - Server's RAM measured in bytes
* `name` - Human readable name of server
* `vnc_password` - VNC Password to connect to server
