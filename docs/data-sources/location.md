--
page_title: "CloudSigma: cloudsigma_library_location"

---

# cloudsigma_library_location

Gets information about a location.


## Example Usage

```hcl
data "cloudsigma_library_location" "frankfurt" {
  filter {
    name   = "id"
    values = ["FRA"]
  }
}
```


## Argument Reference

The following arguments are supported:

* `filter` - (Optional) One or more name/value pairs to use as filters.


## Attributes Reference

In addition to all above arguments, the following attributes are exported:

* `api_endpoint`
* `country_code`
* `display_name`
* `id`
