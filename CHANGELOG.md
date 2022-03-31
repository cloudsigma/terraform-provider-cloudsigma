
<a name="v1.10.1"></a>
## v1.10.1 (2022-03-31)

### Bug Fixes
* **resource/cloudsigma_server:** ensure that meta attribute will be updated when changing


<a name="v1.10.0"></a>
## v1.10.0 (2022-03-28)

### Features
* **resource/cloudsigma_server:** add meta attribute


<a name="v1.9.1"></a>
## v1.9.1 (2022-02-06)

### Bug Fixes
* **server/cloudsigma_server:** change enclave_page_caches attribute type to int

### Maintaining
* update cloudsigma-sdk-go to v0.13.0


<a name="v1.9.0"></a>
## v1.9.0 (2022-01-28)

### Features
* **server/cloudsigma_server:** add enclave_page_caches attribute

### Maintaining
* update cloudsigma-sdk-go to v0.12.0


<a name="v1.8.1"></a>
## v1.8.1 (2022-01-25)

### Bug Fixes
* propagate missing base_url option


<a name="v1.8.0"></a>
## v1.8.0 (2022-01-25)

### Features
* add base_url configuration option

### Maintaining
* update cloudsigma-sdk-go to v0.11.0


<a name="v1.7.2"></a>
## v1.7.2 (2022-01-17)

### Documentation
* update registry documentation for token based authentication


<a name="v1.7.1"></a>
## v1.7.1 (2022-01-17)

### Documentation
* **resource/cloudsigma_server:** add missing documentation for ssh keys attribute

### Features
* add client authorization via bearer token

### Maintaining
* update go version to 1.17 in github worklflows
* update terraform-plugin-sdk to v2.9.0


<a name="v1.6.0"></a>
## v1.6.0 (2021-09-09)

### Features
* **data-source/cloudsigma_drive:** add new data source for drives


<a name="v1.5.1"></a>
## v1.5.1 (2021-08-03)

### Maintaining
* update terraform-plugin-sdk to v2.7.0


<a name="v1.5.0"></a>
## v1.5.0 (2021-08-03)

### Maintaining
* update go version to 1.16 in github worklflows
* add support for darwin_arm64 systems


<a name="v1.4.2"></a>
## v1.4.2 (2021-05-24)

### Maintaining
* update CloudSigma SDK library to version v0.9.0


<a name="v1.4.1"></a>
## v1.4.1 (2021-05-17)

### Bug Fixes
* **resource/cloudsigma_server:** fix tags overiding when attaching drives to servers


<a name="v1.4.0"></a>
## v1.4.0 (2021-05-03)

### Bug Fixes
* **resource/cloudsigma_drive:** change size correctly when drive is mounted
* **resource/cloudsigma_drive:** allow to specify timeout for create operation

### Features
* **resource/cloudsigma_server:** add smp attribute

### Maintaining
* run acceptance tests by release workflow


<a name="v1.3.0"></a>
## v1.3.0 (2021-03-11)

### Features
* **resource/cloudsigma_drive:** add validation for trying to shrink drives size
* **resource/cloudsigma_drive:** add tags attribute
* **resource/cloudsigma_server:** add tags attribute

### Maintaining
* update cloudsigma-sdk-go to v0.7.0


<a name="v1.2.1"></a>
## v1.2.1 (2021-02-18)

### Bug Fixes
* **resource/cloudsigma_server:** revert validate function with diagnostics for ssh_keys attribute

### Documentation
* **resource/cloudsigma_server:** add an example assigning private and public networks


<a name="v1.2.0"></a>
## v1.2.0 (2021-02-17)

### Documentation
* **data-source/cloudsigma_vlan:** fix hcl example
* **resource/cloudsigma_server:** document network field with example

### Features
* **resource/cloudsigma_server:** add vlan_uuid field
* **resource/cloudsigma_server:** add network field

### Maintaining
* update terraform-plugin-sdk to v2.4.3
* **github-actions:** upgrade Go version to 1.15


<a name="v1.1.0"></a>
## v1.1.0 (2020-12-24)

### Features
* **resource/cloudsigma_drive:** replace legacy SchemaValidateFunc with wrapper function
* **resource/cloudsigma_server:** replace legacy SchemaValidateFunc with wrapper function

### Maintaining
* add support for openbsd systems


<a name="v1.0.1"></a>
## v1.0.1 (2020-11-25)

### Bug Fixes
* **resource/cloudsigma_server:** check if server runtime before IP address assignment

### Maintaining
* update terraform-plugin-sdk to v2.3.0


<a name="v1.0.0"></a>
## v1.0.0 (2020-11-09)

### Documentation
* add examples with additional drive for server resource

### Features
* **resource/cloudsigma_drive:** add uuid field
* **resource/cloudsigma_server:** add ipv4_address, ssh_keys fields
* **resource/cloudsigma_server:** add mounted_on field
* **resource/cloudsigma_ssh_key:** add private_key field

### Maintaining
* upgrade terraform-plugin-sdk to v2.2.0


<a name="v0.3.0"></a>
## v0.3.0 (2020-10-08)

### Documentation
* add authentication section

### Features
* **resource/cloudsigma_ssh_key:** add fingerprint field


<a name="v0.2.0"></a>
## v0.2.0 (2020-09-27)

### Features
* **resource/cloudsigma_drive_attachment:** remove drive_attachment resource
* **resource/cloudsigma_server:** add 'drive' option to attach server drives

### Maintaining
* upload draft changelog when creating github release
* add breaking changes notes to unreleased changelog
* remove dependency between lint and test tasks
* upgrade terraform-plugin-sdk to v2.0.3

### BREAKING CHANGE

resource drive_attachment is replaced with internal map in server resource


<a name="v0.1.2"></a>
## v0.1.2 (2020-09-07)

### Bug Fixes
* add missing  prefix by binary file name

### Documentation
* add getting-started guide to documentation


<a name="v0.1.1"></a>
## v0.1.1 (2020-08-28)

### Features
* **resource/cloudsigma_remote_snapshot:** add new resource for remote snapshots

### Maintaining
* run GoReleaser with GitHub actions
* enable GitHub actions
* automate release process with GoReleaser


<a name="v0.1.0"></a>
## v0.1.0 (2020-08-23)

### Maintaining
* generate changelog with git-chglog
* disable non-compile acc tests
* support caching for golanci-lint binary tool

