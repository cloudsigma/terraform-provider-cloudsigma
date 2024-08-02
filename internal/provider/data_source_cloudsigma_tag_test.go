package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceCloudSigmaTag_basic(t *testing.T) {
	var tag cloudsigma.Tag
	tagName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaTagDataSource(tagName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagExists("cloudsigma_tag.ds_foobar_basic", &tag),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_basic", "id"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_basic", "resource_uri"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_basic", "uuid"),
					resource.TestCheckResourceAttr("data.cloudsigma_tag.ds_foobar_basic", "name", tagName),
				),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaTag_uuid(t *testing.T) {
	var tag cloudsigma.Tag
	tagName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaTagDataSourceWithUUID(tagName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagExists("cloudsigma_tag.ds_foobar_uuid", &tag),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_uuid", "id"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_uuid", "resource_uri"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_uuid", "uuid"),
					resource.TestCheckResourceAttr("data.cloudsigma_tag.ds_foobar_uuid", "name", tagName),
				),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaTag_filter(t *testing.T) {
	var tag cloudsigma.Tag
	tagName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaTagDataSourceWithFilter(tagName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagExists("cloudsigma_tag.ds_foobar_filter", &tag),
					resource.TestCheckResourceAttr("data.cloudsigma_tag.ds_foobar_filter", "name", tagName),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_filter", "id"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_filter", "resource_uri"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_tag.ds_foobar_filter", "uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaTag_expectError(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaTagDataSourceWithoutNameAndUUID(),
				ExpectError: regexp.MustCompile(`The attribute "name" or "uuid" must be defined.`),
			},
		},
	})
}

func testAccCloudSigmaTagDataSource(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_tag" "ds_foobar_basic" {
  name = "%s"
}

data "cloudsigma_tag" "ds_foobar_basic" {
  name = cloudsigma_tag.ds_foobar_basic.name
}
`, name)
}

func testAccCloudSigmaTagDataSourceWithUUID(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_tag" "ds_foobar_uuid" {
  name = "%s"
}

data "cloudsigma_tag" "ds_foobar_uuid" {
  uuid = cloudsigma_tag.ds_foobar_uuid.id
}
`, name)
}

func testAccCloudSigmaTagDataSourceWithFilter(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_tag" "ds_foobar_filter" {
  name = "%s"
}

data "cloudsigma_tag" "ds_foobar_filter" {
  filter {
    name   = "name"
    values = [cloudsigma_tag.ds_foobar_filter.name]
  }
}
`, name)
}

func testAccCloudSigmaTagDataSourceWithoutNameAndUUID() string {
	return `
data "cloudsigma_tag" "ds_foobar_without_name_and_uuid" {
  name = ""
  uuid = ""
}
`
}
