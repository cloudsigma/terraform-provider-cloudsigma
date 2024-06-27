# use NVMe (nonvolatile memory express) storage type
resource "cloudsigma_drive" "foobar" {
  media = "disk"
  name  = "foobar"
  size  = 5 * 1024 * 1024 * 1024 # 5GB
  storage_type = "nvme"
}
