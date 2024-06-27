resource "cloudsigma_ssh_key" "admin" {
  name = "admin"
  public_key = file("/Users/terraform/.ssh/id_rsa.pub")
}
