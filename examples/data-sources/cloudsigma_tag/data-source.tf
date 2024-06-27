data "cloudsigma_tag" "my_production_tag" {
  filter {
    name   = "name"
    values = ["production"]
  }
}
