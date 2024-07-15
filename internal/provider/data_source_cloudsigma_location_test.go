package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccDataSourceCloudSigmaLocation_basic(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{{
			Config: testAccCloudSigmaLocationDataSource(),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.cloudsigma_location.zrh", "api_endpoint"),
				resource.TestCheckResourceAttrSet("data.cloudsigma_location.zrh", "country_code"),
				resource.TestCheckResourceAttrSet("data.cloudsigma_location.zrh", "display_name"),
				resource.TestCheckResourceAttrSet("data.cloudsigma_location.zrh", "id"),
				resource.TestCheckResourceAttrSet("data.cloudsigma_location.zrh", "uuid"),
				resource.TestCheckResourceAttr("data.cloudsigma_location.zrh", "id", "ZRH"),
				resource.TestCheckResourceAttr("data.cloudsigma_location.zrh", "uuid", "ZRH"),
			),
		}},
	})
}

func TestAccDataSourceCloudSigmaLocation_expectError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaLocationDataSourceWithoutUUID(),
				ExpectError: regexp.MustCompile(`The attribute "uuid" must be defined.`),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaLocation_upgradeFromSDK(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"cloudsigma": {
						VersionConstraint: "2.1.0",
						Source:            "cloudsigma/cloudsigma",
					},
				},
				Config: testAccCloudSigmaLocationDataSourceWithFilter(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsigma_location.zrh", "id", "ZRH"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccCloudSigmaLocationDataSource(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccCloudSigmaLocationDataSource() string {
	return `
data "cloudsigma_location" "zrh" {
  uuid = "ZRH"
}
`
}

func testAccCloudSigmaLocationDataSourceWithFilter() string {
	return `
data "cloudsigma_location" "zrh" {
  filter {
    name   = "id"
    values = ["ZRH"]
  }
}
`
}

func testAccCloudSigmaLocationDataSourceWithoutUUID() string {
	return `
data "cloudsigma_location" "zrh" {}
`
}
