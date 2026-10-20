package cloudsigma

import (
	"context"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// driveWaitStatus is response-only and must never build mutation payloads.
type driveWaitStatus struct{ UUID, Status string }

func driveStatusRefreshFunc(ctx context.Context, meta interface{}, legacy *cloudsigma.Client, uuid string) retry.StateRefreshFunc {
	clients, ok := meta.(*configClients)
	if !ok || clients == nil || clients.native == nil {
		// Explicit Legacy token/raw-SDK dispatch, never native-error fallback.
		return driveStateRefreshFunc(ctx, legacy, uuid)
	}
	return func() (interface{}, string, error) {
		if clients.basicCredentialError != nil {
			return nil, "", fmt.Errorf("error retrieving drive with UUID %s: %s", uuid, clients.basicCredentialError)
		}
		drive, err := clients.native.DriveWithCallerContext(ctx, uuid)
		if err != nil {
			return nil, "", fmt.Errorf("error retrieving drive with UUID %s: %s", uuid, err)
		}
		status := &driveWaitStatus{UUID: drive.UUID, Status: drive.Status}
		return status, status.Status, nil
	}
}
