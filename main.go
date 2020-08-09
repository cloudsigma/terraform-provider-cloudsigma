package main

import (
	"github.com/cloudsigma/terraform-provider-cloudsigma/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: cloudsigma.Provider,
	})
}
