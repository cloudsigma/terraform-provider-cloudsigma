
<a name="0.3.0"></a>
## 0.3.0 (2020-10-07)

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

