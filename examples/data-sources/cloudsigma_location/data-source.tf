data "cloudsigma_library_location" "frankfurt" {
  filter {
    name   = "id"
    values = ["FRA"]
  }
}
