package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
)

// Keypair is the response-only SSH key projection used by resource refresh.
type Keypair struct {
	PrivateKey    string                 `json:"private_key,omitempty"`
	Fingerprint   string                 `json:"fingerprint,omitempty"`
	HasPrivateKey bool                   `json:"has_private_key,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Name          string                 `json:"name,omitempty"`
	PublicKey     string                 `json:"public_key,omitempty"`
	ResourceURI   string                 `json:"resource_uri,omitempty"`
	UUID          string                 `json:"uuid,omitempty"`
}

// Keypair reads a keypair without automatic OTP exchange or retries.
func (c *Client) Keypair(ctx context.Context, uuid string) (*Keypair, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrives(ctx, keypairEndpoint, "keypairs/"+uuid+"/")
	if err != nil {
		return nil, err
	}
	return v.(*Keypair), nil
}

func decodeKeypair(b []byte) (*Keypair, error) {
	var v Keypair
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
