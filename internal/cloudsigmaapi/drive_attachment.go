package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// PutDriveAttachment sends the exact legacy attachment bytes once.
// Encoding remains at the SDK DTO boundary; no ordinary update policy is implied.
func (c *Client) PutDriveAttachment(ctx context.Context, id string, payload []byte) error {
	if !driveSegment.MatchString(id) {
		return &Error{Category: "transport"}
	}
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	contextFailure := func(err error) error {
		category := "transport"
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			category = "cancelled"
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		return &Error{Category: category}
	}
	if err := ctx.Err(); err != nil {
		return contextFailure(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.base+"drives/"+id+"/", bytes.NewReader(payload))
	if err != nil {
		return &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	req.SetBasicAuth(c.username, c.password)
	resp, err := transport.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return contextFailure(err)
	}
	if resp == nil {
		return &Error{Category: "transport"}
	}
	if resp.StatusCode != http.StatusOK {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	failure := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return failure("unsupported_encoding")
	}
	if resp.Body == nil {
		return failure("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return contextFailure(err)
	}
	if len(body) > responseLimit {
		return failure("response_too_large")
	}
	// Admit one complete document only here, without changing shared GET decoders.
	decoder := json.NewDecoder(bytes.NewReader(body))
	var document json.RawMessage
	if err := decoder.Decode(&document); err != nil {
		return failure("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return failure("decode_error")
	}
	// The caller discards the legacy response model. Do not invent an identity
	// match requirement or project it back into the subsequent request/state.
	var ack map[string]json.RawMessage
	if err := json.Unmarshal(document, &ack); err != nil || ack == nil {
		return failure("decode_error")
	}
	if err := ctx.Err(); err != nil {
		return contextFailure(err)
	}
	return nil
}
