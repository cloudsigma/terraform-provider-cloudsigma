data "cloudsigma_subscription" "static_ip" {
  filter {
    name   = "uuid"
    values = ["35a1d3bb-2fed-4e92-918a-eaf1f6bf3a41"]
  }
}
