package cloudsigma

import (
	"context"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// dispatchServerPower substitutes only the action send, never the lifecycle waiter.
func dispatchServerPower(ctx context.Context, meta interface{}, legacy *cloudsigma.Client, uuid string, start bool) error {
	if clients, ok := meta.(*configClients); ok && clients != nil && clients.native != nil {
		if clients.basicCredentialError != nil {
			return clients.basicCredentialError
		}
		if start {
			return clients.native.StartServer(ctx, uuid)
		}
		return clients.native.StopServer(ctx, uuid)
	}
	var err error
	if start {
		_, _, err = legacy.Servers.Start(ctx, uuid)
	} else {
		_, _, err = legacy.Servers.Stop(ctx, uuid)
	}
	return err
}
