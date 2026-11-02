package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// ServerUpdateRequest is the bounded provider-owned writable core for a server
// definition. Response-only fields (UUID, owner, resource URI, status, jobs,
// runtime and attached resources) are intentionally absent. Drive and NIC
// attachment projections remain on their existing SDK boundary until their
// clear/reference semantics are separately qualified.
type ServerUpdateRequest struct {
	CPU         *int                    `json:"cpu,omitempty"`
	Memory      *int                    `json:"mem,omitempty"`
	Meta        *map[string]interface{} `json:"meta,omitempty"`
	Name        *string                 `json:"name,omitempty"`
	SMP         *int                    `json:"smp,omitempty"`
	VNCPassword *string                 `json:"vnc_password,omitempty"`
}

// UpdateServer performs one bounded synchronous PUT. It admits only HTTP 200,
// decodes one complete JSON response, and requires the response UUID to match
// the requested server. It never retries or follows redirects.
func (c *Client) UpdateServer(ctx context.Context, uuid string, input ServerUpdateRequest) (*Server, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	if err := ctx.Err(); err != nil {
		return nil, remoteContextError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.base+"servers/"+uuid+"/", bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	c.authenticate(req)
	resp, err := transport.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, remoteContextError(ctx, err)
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	fail := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusOK {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return nil, fail(category)
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return nil, fail("unsupported_encoding")
	}
	if resp.Body == nil {
		return nil, fail("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, remoteContextError(ctx, ctxErr)
		}
		return nil, fail("transport")
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, remoteContextError(ctx, ctxErr)
	}
	if len(body) > responseLimit {
		return nil, fail("response_too_large")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	var document json.RawMessage
	if err := dec.Decode(&document); err != nil {
		return nil, fail("decode_error")
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, fail("decode_error")
	}
	var decoded serverResponse
	if err := json.Unmarshal(document, &decoded); err != nil || !driveSegment.MatchString(decoded.UUID) {
		return nil, fail("decode_error")
	}
	if decoded.UUID != uuid {
		return nil, fail("decode_error")
	}
	if err := ctx.Err(); err != nil {
		return nil, remoteContextError(ctx, err)
	}
	return (*Server)(&decoded), nil
}
