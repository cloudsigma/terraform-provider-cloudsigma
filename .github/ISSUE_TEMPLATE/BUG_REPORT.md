---
name: Bug Report
about: Create a report to help improve the terraform-provider-cloudsigma
title: ''
labels: bug
assignees: ''
---

# Bug report

### Description
<!-- Please provide a clear and concise description of the problem you are
facing including the steps to reproduce the issue. -->

<!---
Please note the following potential times when an issue might be in Terraform
core:

* [Configuration Language](https://www.terraform.io/docs/configuration/index.html) or resource ordering issues
* [State](https://www.terraform.io/docs/state/index.html) and [State Backend](https://www.terraform.io/docs/backends/index.html) issues
* [Provisioner](https://www.terraform.io/docs/provisioners/index.html) issues
* [Registry](https://registry.terraform.io/) issues
* Spans resources across multiple providers

If you are running into one of these scenarios, we recommend opening an issue in the
[Terraform core repository](https://github.com/hashicorp/terraform-plugin-sdk/) instead.
--->

### Terraform Version
<!---
Please run `terraform -v` to show the **CloudSigma provider version** as well
as the **Terraform core version**.

If you are not running the latest version of Terraform or the provider, please
upgrade because your issue may have already been fixed.
--->

```
Terraform <TERRAFORM_VERSION>
+ provider.cloudsigma <TERRAFORM_PROVIDER_VERSION>
```

### Affected Resource(s)
<!--- Please list the affected resources and data sources. --->

* cloudsigma_XXXXX

### Terraform Configuration Files
<!--- Information about code formatting: https://help.github.com/articles/basic-writing-and-formatting-syntax/#quoting-code --->

```hcl
# Copy-paste your Terraform configurations here - for large Terraform configs,
# please use a [Github Gist](https://gist.github.com/) instead.
```

### Expected Behavior
<!--- What should have happened? --->

### Actual Behavior
<!--- What actually happened? --->

### Steps to Reproduce
<!--- Please list the steps required to reproduce the issue. --->

1. `terraform apply`

### Debug Output
<!---
Please provide a link to a GitHub Gist containing the complete debug output.
Please do NOT paste the debug output in the issue; just paste a link to the Gist.

To obtain the debug output, define the `TF_LOG=debug` environment variables
before running `terraform apply`.
--->

### Panic Output
<!--- If Terraform produced a panic, please provide a link to a GitHub Gist
containing the output of the `crash.log`. --->


!--- Please keep this note for the community --->
### Community Note

* Please vote on this issue by adding a 👍 [reaction](https://blog.github.com/2016-03-10-add-reactions-to-pull-requests-issues-and-comments/)
  to the original issue to help the community and maintainers prioritize this request
* Please do not leave "+1" or "me too" comments, they generate extra noise for issue followers and do not help prioritize the request
* If you are interested in working on this issue or have submitted a pull request, please leave a comment

<!--- Thank you for keeping this note for the community --->
