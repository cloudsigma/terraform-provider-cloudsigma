package cloudsigma

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// serverPowerStatus is not an SDK Server and must never build mutation payloads.
type serverPowerStatus struct{ UUID, Status string }

func readServerPowerStatus(ctx context.Context, meta interface{}, legacy *cloudsigma.Client, uuid string) (*serverPowerStatus, int, error) {
	if clients, ok := meta.(*configClients); ok && clients != nil && clients.native != nil {
		if clients.basicCredentialError != nil {
			return nil, 0, clients.basicCredentialError
		}
		server, err := clients.native.ServerWithCallerContext(ctx, uuid)
		if err != nil {
			var apiError *cloudsigmaapi.Error
			if errors.As(err, &apiError) {
				return nil, apiError.Status, err
			}
			return nil, 0, err
		}
		return &serverPowerStatus{UUID: server.UUID, Status: server.Status}, 200, nil
	}
	// Explicit Legacy token/raw-SDK dispatch, never native-error fallback.
	server, resp, err := legacy.Servers.Get(ctx, uuid)
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	if err != nil {
		return nil, status, err
	}
	return &serverPowerStatus{UUID: server.UUID, Status: server.Status}, status, nil
}

func serverPowerStateRefreshFunc(ctx context.Context, meta interface{}, legacy *cloudsigma.Client, uuid string) retry.StateRefreshFunc {
	if clients, ok := meta.(*configClients); !ok || clients == nil || clients.native == nil {
		return serverStateRefreshFunc(ctx, legacy, uuid)
	}
	return func() (interface{}, string, error) {
		server, _, err := readServerPowerStatus(ctx, meta, legacy, uuid)
		if err != nil {
			return nil, "", fmt.Errorf("error retrieving server with uuid %s: %s", uuid, err)
		}
		return server, server.Status, nil
	}
}
