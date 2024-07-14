package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceCloudSigmaProfile_basic(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{{
			Config: testAccCloudSigmaProfileDataSource(),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.cloudsigma_profile.me", "id"),
				resource.TestCheckResourceAttrSet("data.cloudsigma_profile.me", "uuid"),
			),
		}},
	})
}

func testAccCloudSigmaProfileDataSource() string {
	return `
data "cloudsigma_profile" "me" {}
`
}
