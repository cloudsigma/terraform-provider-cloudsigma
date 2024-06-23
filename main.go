package main

import (
	"context"
	"flag"
	"log"

	"github.com/cloudsigma/terraform-provider-cloudsigma/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

var (
	version = "dev"
)

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with debug support")
	flag.Parse()

	ctx := context.Background()

	upgradedSDKProvider, err := tf5to6server.UpgradeServer(ctx, cloudsigma.Provider().GRPCProvider)
	if err != nil {
		log.Fatal(err)
	}
	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(provider.New(version)()),
		func() tfprotov6.ProviderServer { return upgradedSDKProvider },
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt
	if debugMode {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve(
		"registry.terraform.io/cloudsigma/cloudsigma",
		muxServer.ProviderServer, serveOpts...,
	)
	if err != nil {
		log.Fatal(err)
	}
}
