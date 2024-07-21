package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceCloudSigmaLibraryDrive_expectError(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaLibraryDriveDataSourceWithoutNameAndUUID(),
				ExpectError: regexp.MustCompile(`The attribute "name" or "uuid" must be defined.`),
			},
		},
	})
}

func testAccCloudSigmaLibraryDriveDataSourceWithoutNameAndUUID() string {
	return `
data "cloudsigma_library_drive" "ds_foobar_without_name_and_uuid" {
  name = ""
  uuid = ""
}
`
}
