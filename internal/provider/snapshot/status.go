package snapshot

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

const (
	snapshotStatusAvailable = "available"
	snapshotStatusCloning   = "cloning_dst"
	snapshotStatusCreating  = "creating"
)

func statusSnapshotStatus(ctx context.Context, client *cloudsigma.Client, snapshotUUID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		snapshot, _, err := client.Snapshots.Get(ctx, snapshotUUID)
		if err != nil {
			return nil, "", fmt.Errorf("unable to get snapshot %s: %w", snapshotUUID, err)
		}
		return snapshot, snapshot.Status, nil
	}
}
