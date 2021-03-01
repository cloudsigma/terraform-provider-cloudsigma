--
page_title: "CloudSigma: cloudsigma_tag"

---

# cloudsigma_tag

Gets information about a Tag.


## Example Usage

```hcl
data "cloudsigma_tag" "my_production_tag" {
  filter = {
    name   = "name"
    values = ["production"]
  }
}
```


## Argument Reference

The following arguments are supported:

* `filter` - (Optional) One or more name/value pairs to use as filters.


## Attributes Reference

In addition to all above arguments, the following attributes are exported:

* `name`
* `resource_uri`
* `uuid`
