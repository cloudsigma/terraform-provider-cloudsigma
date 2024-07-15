data "cloudsigma_location" "frankfurt" {
  filter {
    name   = "id"
    values = ["FRA"]
  }
}
