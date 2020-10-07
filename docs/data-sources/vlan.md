--
page_title: "CloudSigma: cloudsigma_library_vlan"

---

# cloudsigma_library_vlan

Gets information about a location.


## Example Usage

```hcl
data "cloudsigma_library_vlan" "my_vlan" {
  filter = {
    name   = "uuid"
    values = ["10619300-edda-42ba-91e0-7e3df0689d00"]
  }
}
```


## Argument Reference

The following arguments are supported:

* `filter` - (Optional) One or more name/value pairs to use as filters.


## Attributes Reference

In addition to all above arguments, the following attributes are exported:

* `resource_uri`
