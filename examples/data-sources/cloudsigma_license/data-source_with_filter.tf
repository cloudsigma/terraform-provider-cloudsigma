data "cloudsigma_license" "rds" {
  # name for Microsoft Remote Desktop Services (RDS)
  filter {
    name   = "name"
    values = ["msft_6wc_00002"]
  }
}
