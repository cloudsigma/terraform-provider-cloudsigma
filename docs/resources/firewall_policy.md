---
page_title: "CloudSigma: cloudsigma_firewall_policy"

---

# cloudsigma_firewall_policy

Provides a CloudSigma firewall policy resource.


## Example Usage

```hcl
resource "cloudsigma_firewall_policy" "my_policy" {
  name  = "my_policy"
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the firewall policy


## Attributes Reference

In addition to all above arguments, the following attributes are exported:

* `resource_uri`
