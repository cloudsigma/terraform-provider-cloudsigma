package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceCloudSigmaDrive_basic(t *testing.T) {
	var drive cloudsigma.Drive
	driveName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDriveDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaDriveDataSource(driveName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDriveExists("cloudsigma_drive.ds_foobar_basic", &drive),
					resource.TestCheckResourceAttr("data.cloudsigma_drive.ds_foobar_basic", "name", driveName),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_basic", "id"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_basic", "status"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_basic", "storage_type"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_basic", "uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaDrive_name(t *testing.T) {
	var drive cloudsigma.Drive
	driveName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDriveDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaDriveDataSourceWithName(driveName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDriveExists("cloudsigma_drive.ds_foobar_name", &drive),
					resource.TestCheckResourceAttr("data.cloudsigma_drive.ds_foobar_name", "name", driveName),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_name", "id"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_name", "status"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_name", "storage_type"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_name", "uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaDrive_uuid(t *testing.T) {
	var drive cloudsigma.Drive
	driveName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDriveDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaDriveDataSourceWithUUID(driveName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDriveExists("cloudsigma_drive.ds_foobar_uuid", &drive),
					resource.TestCheckResourceAttr("data.cloudsigma_drive.ds_foobar_uuid", "name", driveName),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_uuid", "id"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_uuid", "status"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_uuid", "storage_type"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_uuid", "uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaDrive_filter(t *testing.T) {
	var drive cloudsigma.Drive
	driveName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDriveDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaDriveDataSourceWithFilter(driveName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDriveExists("cloudsigma_drive.ds_foobar_filter", &drive),
					resource.TestCheckResourceAttr("data.cloudsigma_drive.ds_foobar_filter", "name", driveName),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_filter", "id"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_filter", "status"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_filter", "storage_type"),
					resource.TestCheckResourceAttrSet("data.cloudsigma_drive.ds_foobar_filter", "uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceCloudSigmaDrive_expectError(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaDriveDataSourceWithoutNameAndUUID(),
				ExpectError: regexp.MustCompile(`The attribute "name" or "uuid" must be defined.`),
			},
		},
	})
}

func testAccCloudSigmaDriveDataSource(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "ds_foobar_basic" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
}

data "cloudsigma_drive" "ds_foobar_basic" {
  name = cloudsigma_drive.ds_foobar_basic.name
}
`, name)
}

func testAccCloudSigmaDriveDataSourceWithName(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "ds_foobar_name" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
}

data "cloudsigma_drive" "ds_foobar_name" {
  name = cloudsigma_drive.ds_foobar_name.name
}
`, name)
}

func testAccCloudSigmaDriveDataSourceWithUUID(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "ds_foobar_uuid" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
}

data "cloudsigma_drive" "ds_foobar_uuid" {
  uuid = cloudsigma_drive.ds_foobar_uuid.id
}
`, name)
}

func testAccCloudSigmaDriveDataSourceWithFilter(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "ds_foobar_filter" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
}

data "cloudsigma_drive" "ds_foobar_filter" {
  filter {
    name   = "name"
    values = [cloudsigma_drive.ds_foobar_filter.name]
  }
}
`, name)
}

func testAccCloudSigmaDriveDataSourceWithoutNameAndUUID() string {
	return `
data "cloudsigma_drive" "ds_foobar_without_name_and_uuid" {
  name = ""
  uuid = ""
}
`
}
