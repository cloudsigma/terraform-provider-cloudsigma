# Set the variable values in *.tfvars file
# or using -var="cloudsigma_username=..." and -var="cloudsigma_password=..." CLI option
variable "cloudsigma_username" {}
variable "cloudsigma_password" {}

# Configure the CloudSigma Provider
provider "cloudsigma" {
  username = var.cloudsigma_username
  password = var.cloudsigma_password
}

# Create a server
resource "cloudsigma_server" "example" {
  # ...
}
