package cloudsigma

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProvider *schema.Provider
var testAccProviders map[string]*schema.Provider
var testAccProviderFactories func(providers *[]*schema.Provider) map[string]func() (*schema.Provider, error)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"cloudsigma": testAccProvider,
	}
	testAccProviderFactories = func(providers *[]*schema.Provider) map[string]func() (*schema.Provider, error) {
		// SDK v2 compatible hack, the "factory" functions are singletons for the lifecycle of a resource.Test
		var providerNames = []string{"cloudsigma"}
		var factories = make(map[string]func() (*schema.Provider, error), len(providerNames))
		for _, name := range providerNames {
			p := Provider()
			factories[name] = func() (*schema.Provider, error) {
				return p, nil
			}
			*providers = append(*providers, p)
		}
		return factories
	}
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
