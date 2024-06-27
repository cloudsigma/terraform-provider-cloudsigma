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
  vnc_password = "5$zFH9$w"

  network {
    ipv4_address = data.cloudsigma_ip.load_balancer.id
    type         = "static"
  }
}
