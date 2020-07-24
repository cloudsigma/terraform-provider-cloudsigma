---
page_title: "CloudSigma: cloudsigma_acl"

---

# Resource: cloudsigma_acl

Provides a CloudSigma ACL resource. Access Control Lists (ACLs) can be used to
grant permissions to another user to manage your resources. Permissions can be
granted on servers, drives, network resources, and firewall policies.


## Example Usage

```hcl
# Create a new ACL
resource "cloudsigma_acl" "developers" {
  name = "developers"

  permissions = ["ATTACH", "EDIT", "LIST"]
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ACL.
* `permissions` - (Optional) An array of strings containing the permissions.
  This may be "ATTACH", "CLONE", "EDIT", "LIST", "OPEN_VNC", "START", "STOP".


## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the ACL.
* `name` - The name of the ACL.
* `onwer` - The ownership of the ACL.
* `resource_uri` - The resource URI of the ACL.
