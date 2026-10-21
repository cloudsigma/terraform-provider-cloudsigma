package snapshot

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

const (
	snapshotStatusAvailable = "available"
	snapshotStatusCloning   = "cloning_dst"
	snapshotStatusCreating  = "creating"
)

func statusSnapshotStatus(ctx context.Context, client *cloudsigma.Client, snapshotUUID string, native ...*cloudsigmaapi.Client) retry.StateRefreshFunc {
	return func() (any, string, error) {
		if len(native) != 0 && native[0] != nil {
			snap, err := native[0].SnapshotWithCallerContext(ctx, snapshotUUID)
			if err != nil {
				return nil, "", fmt.Errorf("unable to get snapshot %s: %w", snapshotUUID, err)
			}
			return snap, snap.Status, nil
		}
		snapshot, _, err := client.Snapshots.Get(ctx, snapshotUUID)
		if err != nil {
			return nil, "", fmt.Errorf("unable to get snapshot %s: %w", snapshotUUID, err)
		}
		return snapshot, snapshot.Status, nil
	}
}
