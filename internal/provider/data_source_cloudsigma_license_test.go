package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceCloudSigmaLicense_expectError(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaLicenseDataSourceWithoutName(),
				ExpectError: regexp.MustCompile(`The attribute "name" must be defined.`),
			},
		},
	})
}

func testAccCloudSigmaLicenseDataSourceWithoutName() string {
	return `
data "cloudsigma_license" "ds_foobar_without_name" {
  name = ""
}
`
}
