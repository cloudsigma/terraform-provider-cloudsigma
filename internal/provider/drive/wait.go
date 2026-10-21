package drive

import (
	"context"
	"time"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func WaitDriveStatusMountedOrUnmounted(ctx context.Context, client *cloudsigma.Client, driveUUID string, native ...*cloudsigmaapi.Client) error {
	stateConf := retry.StateChangeConf{
		Pending:    []string{driveStatusCloning, driveStatusCreating, driveStatusResizing},
		Target:     []string{driveStatusMounted, driveStatusUnmounted},
		Refresh:    statusDriveStatus(ctx, client, driveUUID, native...),
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return err
	}
	return nil
}
