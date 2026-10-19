package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
)

// Pubkey is the public-only SSH key read projection. It never exposes private key material.
type Pubkey struct {
	Fingerprint   string                 `json:"fingerprint,omitempty"`
	HasPrivateKey bool                   `json:"has_private_key,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Name          string                 `json:"name,omitempty"`
	PublicKey     string                 `json:"public_key,omitempty"`
	ResourceURI   string                 `json:"resource_uri,omitempty"`
	UUID          string                 `json:"uuid,omitempty"`
}

// Pubkey reads the public-only endpoint, without keypair export or OTP exchange.
func (c *Client) Pubkey(ctx context.Context, uuid string) (*Pubkey, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrives(ctx, pubkeyEndpoint, "pubkeys/"+uuid+"/")
	if err != nil {
		return nil, err
	}
	return v.(*Pubkey), nil
}

func decodePubkey(b []byte) (*Pubkey, error) {
	var v Pubkey
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
