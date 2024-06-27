data "cloudsigma_vlan" "my_vlan" {
  filter {
    name   = "uuid"
    values = ["10619300-edda-42ba-91e0-7e3df0689d00"]
  }
}
