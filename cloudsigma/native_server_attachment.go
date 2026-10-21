package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

// putServerCreateAttachment replaces only the Basic CREATE attachment send.
// The pinned SDK DTO remains the encoding boundary, not the Basic transport.
func putServerCreateAttachment(ctx context.Context, meta interface{}, id string, request *cs.ServerUpdateRequest) error {
	legacy, err := extractLegacy(meta)
	if err != nil {
		return err
	}
	if clients, ok := meta.(*configClients); ok && clients.native != nil {
		if clients.basicCredentialError != nil {
			return clients.basicCredentialError
		}
		request.UUID = ""
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(request); err != nil {
			return &cloudsigmaapi.Error{Category: "transport"}
		}
		return clients.native.PutServerAttachment(ctx, id, body.Bytes())
	}
	_, _, err = legacy.Servers.Update(ctx, id, request)
	return err
}
