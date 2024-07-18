data "cloudsigma_drive" "debian" {
  filter {
    name   = "name"
    values = ["Debian 9.13 Server"]
  }
}
