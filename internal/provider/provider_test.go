package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

const accTestPrefix = "tf-acc-test"

var testAccProvider = New("testacc")()
var testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudsigma": providerserver.NewProtocol6WithError(testAccProvider),
}

func TestProviderConfigure_invalidCredentials(t *testing.T) {
	location := os.Getenv("CLOUDSIGMA_LOCATION")
	_ = os.Unsetenv("CLOUDSIGMA_LOCATION")
	defer func() { _ = os.Setenv("CLOUDSIGMA_LOCATION", location) }()

	password := os.Getenv("CLOUDSIGMA_PASSWORD")
	_ = os.Unsetenv("CLOUDSIGMA_PASSWORD")
	defer func() { _ = os.Setenv("CLOUDSIGMA_PASSWORD", password) }()

	token := os.Getenv("CLOUDSIGMA_TOKEN")
	_ = os.Unsetenv("CLOUDSIGMA_TOKEN")
	defer func() { _ = os.Setenv("CLOUDSIGMA_TOKEN", token) }()

	username := os.Getenv("CLOUDSIGMA_USERNAME")
	_ = os.Unsetenv("CLOUDSIGMA_USERNAME")
	defer func() { _ = os.Setenv("CLOUDSIGMA_USERNAME", username) }()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      providerConfigWithoutCredentials,
				ExpectError: regexp.MustCompile("Missing CloudSigma credentials"),
			},
			{
				Config:      providerConfigWithEmptyUsername,
				ExpectError: regexp.MustCompile(`"username" must be set`),
			},
			{
				Config:      providerConfigWithEmptyPassword,
				ExpectError: regexp.MustCompile(`"password" must be set`),
			},
			{
				Config:      providerConfigWithEmptyUsernameAndPassword,
				ExpectError: regexp.MustCompile("Missing CloudSigma credentials"),
			},
			{
				Config:      providerConfigWithUsernamePasswordAndToken,
				ExpectError: regexp.MustCompile("Ambiguous CloudSigma credentials"),
			},
		},
	})
}

func TestProviderUserAgent(t *testing.T) {
	t.Parallel()

	type testCase struct {
		version           string
		expectedUserAgent string
	}
	tests := map[string]testCase{
		"empty_version": {
			version:           "",
			expectedUserAgent: "terraform-provider-cloudsigma/ (+https://registry.terraform.io/providers/cloudsigma/cloudsigma)",
		},
		"dev_version": {
			version:           "dev",
			expectedUserAgent: "terraform-provider-cloudsigma/dev (+https://registry.terraform.io/providers/cloudsigma/cloudsigma)",
		},
		"release_version": {
			version:           "1.1.1",
			expectedUserAgent: "terraform-provider-cloudsigma/1.1.1 (+https://registry.terraform.io/providers/cloudsigma/cloudsigma)",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := &cloudSigmaProvider{version: test.version}
			actualUserAgent := p.userAgent()

			assert.Equal(t, test.expectedUserAgent, actualUserAgent)
		})
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CLOUDSIGMA_LOCATION"); v == "" {
		t.Fatal("CLOUDSIGMA_LOCATION must be set for acceptance tests")
	}

	if v := os.Getenv("CLOUDSIGMA_USERNAME"); v == "" {
		t.Fatal("CLOUDSIGMA_USERNAME must be set for acceptance tests")
	}

	if v := os.Getenv("CLOUDSIGMA_PASSWORD"); v == "" {
		t.Fatal("CLOUDSIGMA_PASSWORD must be set for acceptance tests")
	}
}

const providerConfigWithoutCredentials = `
provider "cloudsigma" {
}
data "cloudsigma_profile" "me" {}
`

const providerConfigWithEmptyUsername = `
provider "cloudsigma" {
  username = ""
  password = "secret-password"
}
data "cloudsigma_profile" "me" {}
`

const providerConfigWithEmptyPassword = `
provider "cloudsigma" {
  username = "username@mail"
  password = ""
}
data "cloudsigma_profile" "me" {}
`

const providerConfigWithEmptyUsernameAndPassword = `
provider "cloudsigma" {
  username = ""
  password = ""
}
data "cloudsigma_profile" "me" {}
`

const providerConfigWithUsernamePasswordAndToken = `
provider "cloudsigma" {
  username = "username@mail"
  password = "secret-password"
  token = "secret-token"
}
data "cloudsigma_profile" "me" {}
`
