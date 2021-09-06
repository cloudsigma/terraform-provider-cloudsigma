--
page_title: "CloudSigma: cloudsigma_drive"

---

# cloudsigma_drive

Gets information about a drive.

## Example Usage

```hcl
data "cloudsigma_drive" "debian" {
  filter = {
    name   = "name"
    values = ["Debian 9.13 Server"]
  }
}
```

## Argument Reference

The following arguments are supported:

- `filter` - (Optional) One or more name/value pairs to use as filters.

## Attributes Reference

In addition to all above arguments, the following attributes are exported:

- `uuid`
- `name`
- `size`
- `status`
- `storage_type`
