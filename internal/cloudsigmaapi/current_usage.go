package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
)

// CurrentUsage preserves the deployment-specific current usage document. The
// account balance and usage resource keys vary by deployment, so callers keep
// the response as JSON fragments instead of imposing a static schema.
type CurrentUsage map[string]json.RawMessage

// CurrentUsage reads the documented singleton current usage object. It is
// read-only and does not perform any planning-time validation or caching.
func (c *Client) CurrentUsage(ctx context.Context) (CurrentUsage, error) {
	v, err := c.getLibraryDrives(ctx, currentUsageEndpoint, "currentusage/")
	if err != nil {
		return nil, err
	}
	return v.(CurrentUsage), nil
}

func decodeCurrentUsage(body []byte) (CurrentUsage, error) {
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&object); err != nil || object == nil {
		return nil, &Error{Category: "decode_error"}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, &Error{Category: "decode_error"}
	}
	return CurrentUsage(object), nil
}
