---
layout: "cloudsigma"
page_title: "CloudSigma: cloudsigma_ssh_key"
sidebar_current: "docs-cloudsigma-resource-ssh-key"
description: |-
  Provides a CloudSigma SSH key resource.
---

# cloudsigma\_ssh_key

 Provides a CloudSigma SSH key resource. to allow you to manage SSH keys. Keys created with this resource can be referenced
 in your server configuration via their ID.


## Example Usage

```hcl
# Create a new SSH key
resource "cloudsigma_ssh_key" "default" {
  name       = "Terraform Example"
  public_key = file("/Users/terraform/.ssh/id_rsa.pub")
}

...
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the SSH key for identification
* `private_key` - (Optional) The private key. If this is a file, it can be read using the file interpolation function
* `public_key` - (Required) The public key. If this is a file, it can be read using the file interpolation function


## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the key
* `name` - The name of the SSH key
* `private_key` - The text of the private key
* `public_key` - The text of the public key
