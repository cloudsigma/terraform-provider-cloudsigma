package cloudsigma

import (
	"context"
	"encoding/json"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

// createServerDefinition substitutes only the Basic create send. The complete
// response model still feeds the existing SDK attachment PUT and lifecycle.
func createServerDefinition(ctx context.Context, meta interface{}, request *cs.ServerCreateRequest) (*cs.Server, error) {
	legacy, err := extractLegacy(meta)
	if err != nil {
		return nil, err
	}
	if clients, ok := meta.(*configClients); ok && clients.native != nil {
		if clients.basicCredentialError != nil {
			return nil, clients.basicCredentialError
		}
		wire, err := json.Marshal(request.Servers[0])
		if err != nil {
			return nil, err
		}
		var input cloudsigmaapi.ServerCreateRequest
		if err := json.Unmarshal(wire, &input); err != nil {
			return nil, err
		}
		result, createErr := clients.native.CreateServer(ctx, input)
		if result == nil {
			return nil, createErr
		}
		wire, err = json.Marshal(result)
		if err != nil {
			return nil, err
		}
		var server cs.Server
		if err := json.Unmarshal(wire, &server); err != nil {
			return nil, err
		}
		server.Jobs = append(json.RawMessage(nil), result.Jobs...)
		return &server, createErr
	}
	servers, _, err := legacy.Servers.Create(ctx, request)
	if err != nil {
		return nil, err
	}
	return &servers[0], nil
}
