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

  ssh_keys = ["ssh-key-uuid"]
  tags = [
    "first-tag-uuid",
    "second-tag-uuid",
  ]
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
    values = ["33.44.55.66"]
  }
}

resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "cloudsigma"

  network {
    ipv4_address = data.cloudsigma_ip.load_balancer.id
    type         = "static"
  }
}
```

### With static IP address and private VLAN

```hcl
resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "cloudsigma"

  network {
    ipv4_address = "33.44.55.66"
    type = "static"
  }
  network {
    vlan_uuid = "10619300-edda-42ba-91e0-7e3df0689d00"
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
* `ssh_keys` - (Optional) A list of the SSH key UUIDs to be applied to the server
* `tags` - (Optional) A list of the tags UUIDs to be applied to the server
* `smp` - (Optional) Symmetric Multiprocessing (SMP) i.e. number of CPU cores
* `meta` - (Optional) The field can be used to store arbitrary information in key-value form.

## Attributes Reference

The following attributes are exported:

* `cpu` - Server's CPU Clock speed measured in MHz
* `memory` - Server's RAM measured in bytes
* `name` - Human readable name of server
* `vnc_password` - VNC Password to connect to server
