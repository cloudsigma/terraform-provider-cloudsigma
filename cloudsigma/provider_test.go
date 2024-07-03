package cloudsigma

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider"
)

var testAccProto6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudsigma": func() (tfprotov6.ProviderServer, error) {
		ctx := context.Background()
		upgradedSDKProvider, err := tf5to6server.UpgradeServer(ctx, Provider().GRPCProvider)
		if err != nil {
			return nil, err
		}
		providers := []func() tfprotov6.ProviderServer{
			func() tfprotov6.ProviderServer { return upgradedSDKProvider },
			providerserver.NewProtocol6(provider.New("testacc")()),
		}
		muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
		if err != nil {
			return nil, err
		}
		return muxServer.ProviderServer(), nil
	},
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CLOUDSIGMA_USERNAME"); v == "" {
		t.Fatal("CLOUDSIGMA_USERNAME must be set for acceptance tests")
	}

	if v := os.Getenv("CLOUDSIGMA_PASSWORD"); v == "" {
		t.Fatal("CLOUDSIGMA_PASSWORD must be set for acceptance tests")
	}
}

func sharedClient() (*cloudsigma.Client, error) {
	location := os.Getenv("CLOUDSIGMA_LOCATION")
	if location == "" {
		return nil, fmt.Errorf("empty CLOUDSIGMA_LOCATION")
	}

	username := os.Getenv("CLOUDSIGMA_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("CLOUDSIGMA_USERNAME must be set for acceptance tests")
	}

	password := os.Getenv("CLOUDSIGMA_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("CLOUDSIGMA_PASSWORD must be set for acceptance tests")
	}

	opts := []cloudsigma.ClientOption{cloudsigma.WithUserAgent("terraform-provider-cloudsigma/sweeper")}
	opts = append(opts, cloudsigma.WithLocation(location))
	creds := cloudsigma.NewUsernamePasswordCredentialsProvider(username, password)
	return cloudsigma.NewClient(creds, opts...), nil
}
