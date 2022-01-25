---
page_title: "CloudSigma: Getting Started with CloudSigma provider"
description: |-
  Getting started with the CloudSigma provider
---

# Getting Started with the CloudSigma Provider

## Before you begin

* Be sure that you can create resources in the CloudSigma location you use to
[log in](https://zrh.cloudsigma.com/ui/4.0/login).
* [Install Terraform](https://www.terraform.io/intro/getting-started/install.html)
and read the Terraform getting started guide that follows. This guide will
assume basic proficiency with Terraform - it is an introduction to the CloudSigma
provider.

## Configuring the Provider

Create a Terraform config file named `"main.tf"`. Inside, you'll want to include
the following configuration:

```hcl
provider "cloudsigma" {
  username = var.cloudsigma_username
  password = var.cloudsigma_password
}
```

* The `username` field should be your user email.
* The `password` field is your password.
* The `location` field is optional (default value is `zrh`). If you want to try
another location, check [available locations](https://docs.cloudsigma.com/en/latest/general.html#api-endpoint)
and use the location code as value.
* The `base_url` field is optional (default value is `cloudsigma.com/api/2.0/`).

## Creating a Server
Add the following to your config file:

```hcl
resource "cloudsigma_server" "web" {
  cpu          = 2000              # 2GHz CPU
  memory       = 512 * 1024 * 1024 # 512MB RAM
  name         = "web"
  vnc_password = "cloudsigma"
}
```

With a Terraform config and with your credentials configured, it's time to
provision your resources:

```sh
terraform apply
```

Congratulations! You've gotten started using the CloudSigma provider and provisioned
a virtual machine on CloudSigma Cloud.

Run `terraform destroy` to tear down your resources.

## Conclusion

Terraform offers you an effective way to manage CloudSigma resources. Check out
the extensive documentation of the CloudSigma provider linked from the menu.
