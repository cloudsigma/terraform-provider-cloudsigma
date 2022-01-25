---
page_title: "Provider: CloudSigma"
description: |-
  The CloudSigma provider is used to interact with the many resources supported by CloudSigma through its APIs.
---

# CloudSigma Provider

The CloudSigma provider is used to interact with the resources supported by
CloudSigma. The provider needs to be configured with proper credentials before
it can be used.

Use the navigation to the left to read about the available resources.


## Example Usage

```hcl
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
```


## Authentication

The CloudSigma authentication is based on HTTP Basic Authentication with an
user email as an **username** and a **password**.

The CloudSigma provider offers two ways of providing these credentials. The
following methods are supported, in this priority order:

1. [Static credentials](#static-credentials)
2. [Environment variables](#environment-variables)

### Static credentials

!> **Warning:** Hard-coding credentials into any Terraform configuration is not
recommended, and risks secret leakage should this file ever be committed to a
public version control system. We recommend to use `terraform.tfvars` file and
add this file to your .gitignore.

Static credentials can be provided by adding `username` and `password` attributes
in-line in the CloudSigma provider block.

```hcl
provider "cloudsigma" {
  username = "my-email"
  password = "my-password"
}
```

### Environment variables

You can provide your credentials via the `CLOUDSIGMA_TOKEN`, `CLOUDSIGMA_USERNAME`,
`CLOUDSIGMA_PASSWORD` environment variables.

```hcl
provider "cloudsigma" {}
```

Usage:

```bash
$ export CLOUDSIGMA_USERNAME="my-email"
$ export CLOUDSIGMA_PASSWORD="my-password"
$ terraform plan
```


## Argument Reference

The following arguments are supported:

* `token` - (required) Your CloudSigma access token. Alternatively, this can also
  be specified using an environment variable called `CLOUDSIGMA_TOKEN`.
* `username` - (Required) Your CloudSigma email address. Alternatively, this can
  also be specified using an environment variable called `CLOUDSIGMA_USERNAME`.
* `password` - (Required) Your CloudSigma password. Alternatively, this can
  also be specified using an environment variable called `CLOUDSIGMA_PASSWORD`.
* `location` - (Optional) This can be used to override the location for
  CloudSigma API requests (Defaults to the value of the `CLOUDSIGMA_LOCATION`
  environment variable or `zrh` if unset).
* `base_url` - (Optional) This is the base URL for for CloudSigma API requests.
  (Defaults to the value of the `CLOUDSIGMA_BASE_URL` environment variable or
  `cloudsigma.com/api/2.0/` if unset). The value must end with a slash.
