---
layout: "cloudsigma"
page_title: "Provider: CloudSigma"
sidebar_current: "docs-cloudsigma-index"
description: |-
  The CloudSigma provider is used to interact with the many resources supported by CloudSigma through its APIs.
---

# CloudSigma Provider

The CloudSigma provider is used to interact with the resources supported by CloudSigma. The provider needs to be configured
with proper credentials before it can be used.

Use the navigation to the left to read about the available resources.


## Example Usage

```hcl
# Set the variable value in *.tfvars file
# or using -var="cloudsigma_username=..." and -var="cloudsigma_password=..." CLI option
# `CLOUDSIGMA_USERNAME` instead.

variable "cloudsigma_username" {}
variable "cloudsigma_password" {}

provider "cloudsigma" {
  username = var.cloudsigma_username
  password = var.cloudsigma_password

  # whilst the `version` attribute is optional, we recommend pinning to a given version of the Provider
  version = "=1.0.0"
}

...
```


## Argument Reference

The following arguments are supported:

* `username` - (Required) Your Cloudsigma email address. It can also be specified as a shell environment variable called `CLOUDSIGMA_USERNAME`.
* `password` - (Required) Your Cloudsigma password. It can also be specified as a shell environment variable called `CLOUDSIGMA_PASSWORD`.
* `location` - (Optional) This can be used to override the location for CloudSigma API requests (Defaults to the value
  of the `CLOUDSIGMA_LOCATION` environment variable or `https://zrh.cloudsigma.com/api/2.0/` if unset).
