package snapshot

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func WaitSnapshotStatusAvailable(ctx context.Context, client *cloudsigma.Client, snapshotUUID string) error {
	stateConf := retry.StateChangeConf{
		Pending:    []string{snapshotStatusCloning, snapshotStatusCreating},
		Target:     []string{snapshotStatusAvailable},
		Refresh:    statusSnapshotStatus(ctx, client, snapshotUUID),
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return err
	}
	return nil
}
