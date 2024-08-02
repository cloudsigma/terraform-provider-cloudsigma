package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func testAccCheckDriveDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := sharedClient("testacc")
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudsigma_drive" {
			continue
		}

		drive, _, err := client.Drives.Get(ctx, rs.Primary.ID)
		if err == nil && drive.UUID == rs.Primary.ID {
			return fmt.Errorf("drive (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckDriveExists(n string, drive *cloudsigma.Drive) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("no drive ID set")
		}

		ctx := context.Background()
		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		retrievedDrive, _, err := client.Drives.Get(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("could not get drive: %s", err)
		}

		if retrievedDrive.UUID != rs.Primary.ID {
			return errors.New("drive not found")
		}

		drive = retrievedDrive
		return nil
	}
}
