resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "5$zFH9$w"

  network {
    ipv4_address = "33.44.55.66"
    type         = "static"
  }
  network {
    vlan_uuid = "<vlan-uuid>"
  }
}
