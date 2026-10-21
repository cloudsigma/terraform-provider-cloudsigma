package drive

import (
	"context"
	"fmt"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

const (
	driveStatusCloning     = "cloning_dst"
	driveStatusCreating    = "creating"
	driveStatusMounted     = "mounted"
	driveStatusResizing    = "resizing"
	driveStatusUnavailable = "unavailable"
	driveStatusUnmounted   = "unmounted"
)

func statusDriveStatus(ctx context.Context, client *cloudsigma.Client, driveUUID string, native ...*cloudsigmaapi.Client) retry.StateRefreshFunc {
	return func() (any, string, error) {
		if len(native) > 0 && native[0] != nil {
			drive, err := native[0].DriveWithCallerContext(ctx, driveUUID)
			if err != nil {
				return nil, "", fmt.Errorf("unable to get drive %s: %w", driveUUID, err)
			}
			if drive.Status == driveStatusUnavailable {
				tflog.Warn(ctx, "Drive status is unavailable")
			}
			return drive, drive.Status, nil
		}
		drive, _, err := client.Drives.Get(ctx, driveUUID)
		if err != nil {
			return nil, "", fmt.Errorf("unable to get drive %s: %w", driveUUID, err)
		}
		if drive.Status == driveStatusUnavailable {
			tflog.Warn(ctx, "Drive status is unavailable")
		}
		return drive, drive.Status, nil
	}
}
