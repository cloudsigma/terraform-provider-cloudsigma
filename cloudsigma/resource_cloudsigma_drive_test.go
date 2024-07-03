package cloudsigma

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestAccCloudSigmaDrive_basic(t *testing.T) {
	var drive cloudsigma.Drive
	driveName := fmt.Sprintf("tf-acc-test--%s", acctest.RandString(10))
	tagName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProto6ProviderFactories,
		CheckDestroy:             testAccCheckCloudSigmaDriveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaDriveConfig_basic(driveName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "media", "disk"),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "name", driveName),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "size", "5368709120"),
					resource.TestCheckResourceAttrSet("cloudsigma_drive.test", "resource_uri"),
					resource.TestCheckResourceAttrSet("cloudsigma_drive.test", "uuid"),
				),
			},
			{
				Config: testAccCloudSigmaDriveConfig_addTag(tagName, driveName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "tags.#", "1"),
				),
			},
			{
				Config: testAccCloudSigmaDriveConfig_noTag(driveName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccCloudSigmaDrive_emptyTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProto6ProviderFactories,
		CheckDestroy:             testAccCheckCloudSigmaDriveDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaDriveConfig_emptyTag(),
				ExpectError: regexp.MustCompile("tags.* must not be empty, got"),
			},
		},
	})
}

func TestAccCloudSigmaDrive_changeSize(t *testing.T) {
	var drive cloudsigma.Drive
	driveName := fmt.Sprintf("tf-acc-test--%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProto6ProviderFactories,
		CheckDestroy:             testAccCheckCloudSigmaDriveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaDriveConfig_basic(driveName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "media", "disk"),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "name", driveName),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "size", "5368709120"),
					resource.TestCheckResourceAttrSet("cloudsigma_drive.test", "resource_uri"),
					resource.TestCheckResourceAttrSet("cloudsigma_drive.test", "uuid"),
				),
			},
			{
				Config: testAccCloudSigmaDriveConfig_changeSize(driveName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "size", "16106127360"),
				),
			},
		},
	})
}

func TestAccCloudSigmaDrive_changeStorageType(t *testing.T) {
	var drive cloudsigma.Drive
	driveName := fmt.Sprintf("tf-acc-test--%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProto6ProviderFactories,
		CheckDestroy:             testAccCheckCloudSigmaDriveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaDriveConfig_storageType(driveName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "storage_type", "dssd"),
				),
			},
			{
				Config:      testAccCloudSigmaDriveConfig_changeStorageType(driveName),
				ExpectError: regexp.MustCompile("drives `storage_type` cannot be changed after creation.*"),
			},
		},
	})
}

func testAccCheckCloudSigmaDriveDestroy(s *terraform.State) error {
	client, err := sharedClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudsigma_drive" {
			continue
		}

		drive, _, err := client.Drives.Get(context.Background(), rs.Primary.ID)
		if err == nil && drive.UUID == rs.Primary.ID {
			return fmt.Errorf("drive (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckCloudSigmaDriveExists(n string, drive *cloudsigma.Drive) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no drive ID is set")
		}

		client, err := sharedClient()
		if err != nil {
			return err
		}
		retrievedDrive, _, err := client.Drives.Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if retrievedDrive.UUID != rs.Primary.ID {
			return fmt.Errorf("drive not found")
		}

		*drive = *retrievedDrive
		return nil
	}
}

func testAccCloudSigmaDriveConfig_basic(driveName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
}
`, driveName)
}

func testAccCloudSigmaDriveConfig_addTag(tagName, driveName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_tag" "test" {
  name = "%s"
}

resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024

  tags = [cloudsigma_tag.test.id]
}
`, tagName, driveName)
}

func testAccCloudSigmaDriveConfig_noTag(driveName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024

  tags = []
}
`, driveName)
}

func testAccCloudSigmaDriveConfig_emptyTag() string {
	return `
resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "drive-with-invalid-empty-tag-element"
  size  = 5 * 1024 * 1024 * 1024

  tags = [""]
}
`
}

func testAccCloudSigmaDriveConfig_changeSize(driverName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "test" {
  media = "disk"
  name = "%s"
  size = 15 * 1024 * 1024 * 1024
}
`, driverName)
}

func testAccCloudSigmaDriveConfig_storageType(driveName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
  storage_type = "dssd"
}
`, driveName)
}

func testAccCloudSigmaDriveConfig_changeStorageType(driveName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
  storage_type = "zadara"
}
`, driveName)
}
