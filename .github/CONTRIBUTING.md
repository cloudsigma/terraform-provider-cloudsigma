# Contributing to Terraform - CloudSigma Provider


First: if you're unsure or afraid of anything, ask for help! You can submit a work in progress (WIP) pull request, or file
an issue with the parts you know. We'll do our best to guide you in the right direction, and let you know if there are
guidelines we will need to follow. We want people to be able to participate without fear of doing the wrong thing.


## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (please check
the [requirements](https://github.com/terraform-providers/terraform-provider-cloudsigma#requirements) before proceeding).

*Note:* This project uses [Go Modules](https://blog.golang.org/using-go-modules) making it safe to work with it outside
of your existing [GOPATH](http://golang.org/doc/code.html#GOPATH). The instructions that follow assume a directory in your
home directory outside of the standard GOPATH (i.e `$HOME/development/terraform-providers/`).

Clone repository to: `$HOME/development/terraform-providers/`

```sh
$ mkdir -p $HOME/development/terraform-providers/; cd $HOME/development/terraform-providers/
$ git clone git@github.com:terraform-providers/terraform-provider-cloudsigma
...
```

Enter the provider directory and run `make tools`. This will install the needed tools for the provider.

```sh
$ make tools
```

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `build` directory.

```sh
$ make build
...
$ build/terraform-provider-cloudsigma
...
```


## Using the Provider

To use a released provider in your Terraform environment, run [`terraform init`](https://www.terraform.io/docs/commands/init.html)
and Terraform will automatically install the provider. To specify a particular provider version when installing released
providers, see the [Terraform documentation on provider versioning](https://www.terraform.io/docs/configuration/providers.html#version-provider-versions).

To instead use a custom-built provider in your Terraform environment (e.g. the provider binary from the build instructions
above), follow the instructions to [install it as a plugin.](https://www.terraform.io/docs/plugins/basics.html#installing-plugins)
After placing the custom-built provider into your plugins directory,  run `terraform init` to initialize it.

For either installation method, documentation about the provider specific configuration options can be found on the [provider's website](https://www.terraform.io/docs/providers/cloudsigma/index.html).


## Testing the Provider

In order to test the provider, you can run `make test`.

*Note:* Make sure no `CLOUDSIGMA_USERNAME` and `CLOUDSIGMA_PASSWORD` variables are set.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
