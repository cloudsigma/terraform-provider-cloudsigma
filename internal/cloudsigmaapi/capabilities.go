package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
)

// Capabilities preserves the deployment-specific capability document without
// imposing a fixed schema on dynamic location, quota, and feature fields.
// Values remain JSON fragments so callers can apply explicit, account-scoped
// validation without hidden plan-time network calls.
type Capabilities map[string]json.RawMessage

// Capabilities reads the documented dynamic capability object. The explicit
// limit=0 query requests the complete document where the deployment supports
// pagination-compatible query handling.
func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	v, err := c.getLibraryDrives(ctx, capabilitiesEndpoint, "capabilities/?limit=0")
	if err != nil {
		return nil, err
	}
	return v.(Capabilities), nil
}

func decodeCapabilities(body []byte) (Capabilities, error) {
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&object); err != nil || object == nil {
		return nil, &Error{Category: "decode_error"}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, &Error{Category: "decode_error"}
	}
	return Capabilities(object), nil
}
