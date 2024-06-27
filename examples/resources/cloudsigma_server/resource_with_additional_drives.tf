resource "cloudsigma_drive" "data" {
  media = "disk"
  name  = "web-data"
  size  = 5 * 1024 * 1024 * 1024 # 5GB
}

resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "5$zFH9$w"

  drive {
    uuid = cloudsigma_drive.data.id
  }
}
