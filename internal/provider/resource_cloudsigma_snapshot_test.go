package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func init() {
	resource.AddTestSweepers("cloudsigma_snapshot", &resource.Sweeper{
		Name: "cloudsigma_snapshot",
		F:    testSweepSnapshots,
	})
}

func testSweepSnapshots(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	snapshots, _, err := client.Snapshots.List(ctx)
	if err != nil {
		return fmt.Errorf("getting snapshot list: %w", err)
	}

	for _, snapshot := range snapshots {
		if strings.HasPrefix(snapshot.Name, accTestPrefix) {
			slog.Info("Deleting cloudsigma_snapshot", "name", snapshot.Name, "uuid", snapshot.UUID)
			_, err := client.Snapshots.Delete(ctx, snapshot.UUID)
			if err != nil {
				slog.Warn("Error deleting snapshot during sweep", "name", snapshot.Name, "error", err)
			}
		}
	}

	return nil
}

func TestAccResourceCloudSigmaSnapshot_basic(t *testing.T) {
	var snapshot cloudsigma.Snapshot
	snapshotName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSnapshotDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaSnapshotResource(snapshotName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnapshotExists("cloudsigma_snapshot.r_foobar_basic", &snapshot),
					resource.TestCheckResourceAttr("cloudsigma_snapshot.r_foobar_basic", "name", snapshotName),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_basic", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_basic", "resource_uri"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_basic", "status"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_basic", "timestamp"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_basic", "uuid"),
				),
			},
		},
	})
}

func TestAccResourceCloudSigmaSnapshot_update(t *testing.T) {
	var snapshot cloudsigma.Snapshot
	driveName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	snapshotName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	snapshotNameUpdated := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSnapshotDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaSnapshotResourceForUpdate(driveName, snapshotName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnapshotExists("cloudsigma_snapshot.r_foobar_update", &snapshot),
					resource.TestCheckResourceAttr("cloudsigma_snapshot.r_foobar_update", "name", snapshotName),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "resource_uri"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "status"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "timestamp"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "uuid"),
				),
			},
			{
				Config: testAccCloudSigmaSnapshotResourceForUpdate(driveName, snapshotNameUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnapshotExists("cloudsigma_snapshot.r_foobar_update", &snapshot),
					resource.TestCheckResourceAttr("cloudsigma_snapshot.r_foobar_update", "name", snapshotNameUpdated),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "resource_uri"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "status"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "timestamp"),
					resource.TestCheckResourceAttrSet("cloudsigma_snapshot.r_foobar_update", "uuid"),
				),
			},
		},
	})
}

func TestAccResourceCloudSigmaSnapshot_expectError(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSnapshotDestroy,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaSnapshotResourceWithoutDrive(),
				ExpectError: regexp.MustCompile(`The argument "drive" is required`),
			},
			{
				Config:      testAccCloudSigmaSnapshotResourceWithoutName(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
		},
	})
}

func TestAccResourceCloudSigmaSnapshot_upgradeFromSDK(t *testing.T) {
	snapshotName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckSnapshotDestroy,

		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"cloudsigma": {
						VersionConstraint: "2.1.0",
						Source:            "cloudsigma/cloudsigma",
					},
				},
				Config: testAccCloudSigmaSnapshotResource(snapshotName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsigma_snapshot.r_foobar_basic", "name", snapshotName),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccCloudSigmaSnapshotResource(snapshotName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccCheckSnapshotDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := sharedClient("testacc")
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudsigma_snapshot" {
			continue
		}

		snapshot, _, err := client.Snapshots.Get(ctx, rs.Primary.ID)
		if err == nil && snapshot.UUID == rs.Primary.ID {
			return fmt.Errorf("snapshot (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckSnapshotExists(n string, snapshot *cloudsigma.Snapshot) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("no snapshot ID set")
		}

		ctx := context.Background()
		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		retrievedSnapshot, _, err := client.Snapshots.Get(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("could not get snapshot: %s", err)
		}

		if retrievedSnapshot.UUID != rs.Primary.ID {
			return errors.New("snapshot not found")
		}

		snapshot = retrievedSnapshot
		return nil
	}
}

func testAccCloudSigmaSnapshotResource(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "r_foobar_basic" {
  media = "disk"
  name = "%[1]s"
  size = 5 * 1024 * 1024 * 1024
}

resource "cloudsigma_snapshot" "r_foobar_basic" {
  drive = cloudsigma_drive.r_foobar_basic.uuid
  name = "%[1]s"
}`, name)
}

func testAccCloudSigmaSnapshotResourceForUpdate(driveName, snapshotName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_drive" "r_foobar_update" {
  media = "disk"
  name = "%s"
  size = 6 * 1024 * 1024 * 1024
}

resource "cloudsigma_snapshot" "r_foobar_update" {
  drive = cloudsigma_drive.r_foobar_update.uuid
  name = "%s"
}`, driveName, snapshotName)
}

func testAccCloudSigmaSnapshotResourceWithoutDrive() string {
	return `
resource "cloudsigma_snapshot" "r_foobar_without_name" {
  name = "r_foobar_without_drive"
}
`
}

func testAccCloudSigmaSnapshotResourceWithoutName() string {
	return `
resource "cloudsigma_snapshot" "r_foobar_without_name" {
  drive = "32a65937-2bee-4c60-9ab1-2198504f5d0e"
}
`
}
