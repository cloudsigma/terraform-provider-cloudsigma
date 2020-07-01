---
layout: "cloudsigma"
page_title: "CloudSigma: cloudsigma_tag_key"
sidebar_current: "docs-cloudsigma-resource-tag-key"
description: |-
  Provides a CloudSigma tag resource.
---

# cloudsigma\_tag

Provides a CloudSigma tag resource. A Tag is a label that can be applied to a CloudSigma resource in order to better
organize or facilitate the lookups and actions on it. Tags created with this resource can be referenced in your
configurations via their ID.


## Example Usage

```hcl
# Create a new tag
resource "cloudsigma_tag" "production" {
  name = "production"
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the tag


## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the tag.
* `name` - The name of the tag.
* `onwer` - The ownership of the tag.
* `resource_uri` - The resource URI of the tag.
