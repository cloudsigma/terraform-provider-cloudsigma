package cloudsigma

import (
	"context"
	"encoding/json"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// driveForTagAttachment migrates the read only. The complete pinned SDK model
// still feeds the legacy PUT; this adapter does not authorize new writable fields.
func driveForTagAttachment(ctx context.Context, meta interface{}, id string) (*cloudsigma.Drive, error) {
	client, err := extractLegacy(meta)
	if err != nil {
		return nil, err
	}
	if clients, ok := meta.(*configClients); ok && clients.native != nil {
		if clients.basicCredentialError != nil {
			return nil, clients.basicCredentialError
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		drive, err := clients.native.DriveWithCallerContext(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Both models have identical wire fields. Preserve the full response rather
		// than constructing a partial model that silently drops PUT fields.
		wire, err := json.Marshal(drive)
		if err != nil {
			return nil, err
		}
		var result cloudsigma.Drive
		if err := json.Unmarshal(wire, &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	drive, _, err := client.Drives.Get(ctx, id)
	return drive, err
}
