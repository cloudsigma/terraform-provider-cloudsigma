package cloudsigma

import (
	"os"
	"testing"
)

// var testAccProviders map[string]terraform.ResourceProvider
// var testAccProvider *schema.Provider
//
// func init() {
// 	testAccProvider = Provider().(*schema.Provider)
// 	testAccProviders = map[string]terraform.ResourceProvider{
// 		"cloudsigma": testAccProvider,
// 	}
// }

// func TestProvider(t *testing.T) {
// 	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
// 		t.Fatalf("err: %s", err)
// 	}
// }

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
