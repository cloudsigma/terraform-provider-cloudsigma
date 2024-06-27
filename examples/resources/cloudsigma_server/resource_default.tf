resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 521 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "5$zFH9$w"

  ssh_keys = ["<ssh-key-uuid>"]
  tags     = ["<first-tag-uuid>", "<second-tag-uuid>"]
}
