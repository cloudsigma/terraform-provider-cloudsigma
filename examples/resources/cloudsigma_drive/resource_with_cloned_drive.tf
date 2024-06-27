data "cloudsigma_library_drive" "ubuntu" {
  filter {
    name   = "name"
    values = ["Ubuntu 22.04 LTS"]
  }
}

resource "cloudsigma_drive" "foobar" {
  media = "disk"
  name  = "foobar"
  size  = 5 * 1024 * 1024 * 1024 # 5GB

  clone_drive_id = data.cloudsigma_library_drive.ubuntu.id
}
