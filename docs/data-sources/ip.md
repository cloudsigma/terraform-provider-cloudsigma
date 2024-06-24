---
page_title: "CloudSigma: cloudsigma_ip"

---

# cloudsigma_ip

Gets information about an IP.


## Example Usage

```hcl
data "cloudsigma_ip" "my_ip" {
  filter = {
    name   = "uuid"
    values = ["0.0.0.0"]
  }
}
```


## Argument Reference

The following arguments are supported:

* `filter` - (Optional) One or more name/value pairs to use as filters.


## Attributes Reference

In addition to all above arguments, the following attributes are exported:

* `gateway`
* `netmask`
* `resource_uri` - The resource URI of the IP.
