package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceCloudSigmaSubscription_expectError(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaSubscriptionDataSourceWithoutUUID(),
				ExpectError: regexp.MustCompile(`The attribute "uuid" must be defined.`),
			},
		},
	})
}

func testAccCloudSigmaSubscriptionDataSourceWithoutUUID() string {
	return `
data "cloudsigma_subscription" "ds_foobar_without_uuid" {
  uuid = ""
}
`
}
