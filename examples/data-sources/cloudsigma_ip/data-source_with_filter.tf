data "cloudsigma_ip" "my_ip" {
  filter {
    name   = "uuid"
    values = ["0.0.0.0"]
  }
}
