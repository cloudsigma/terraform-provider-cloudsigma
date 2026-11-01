package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// TagCreateRequest contains only the writable fields used by the provider's
// tag resource. Response-only owner, resource_uri, uuid, and resources fields
// are never projected into a create request.
type TagCreateRequest struct {
	Name string `json:"name,omitempty"`
}

// TagUpdateRequest contains only fields the provider currently manages.
type TagUpdateRequest struct {
	Name string `json:"name,omitempty"`
}

// Tags preserves the existing first-page tags/ request without adding query parameters.
func (c *Client) Tags(ctx context.Context) ([]Tag, error) {
	v, e := c.get(ctx, tagsEndpoint)
	if e != nil {
		return nil, e
	}
	return v.([]Tag), nil
}

// Tag reads one tag by UUID using the documented /tags/{uuid}/ route.
func (c *Client) Tag(ctx context.Context, uuid string) (*Tag, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrives(ctx, tagsEndpoint, "tags/"+uuid+"/")
	if err != nil {
		return nil, err
	}
	tag := v.(*Tag)
	if !driveSegment.MatchString(tag.UUID) {
		return nil, &Error{Category: "decode_error"}
	}
	return tag, nil
}

// CreateTag performs one documented POST and validates one acknowledged tag.
func (c *Client) CreateTag(ctx context.Context, input TagCreateRequest) (*Tag, error) {
	payload, err := json.Marshal(struct {
		Objects []TagCreateRequest `json:"objects"`
	}{Objects: []TagCreateRequest{input}})
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	var document struct {
		Objects []Tag `json:"objects"`
	}
	if err := c.doTagWrite(ctx, http.MethodPost, "tags/", payload, http.StatusCreated, &document); err != nil {
		return nil, err
	}
	if len(document.Objects) != 1 || !driveSegment.MatchString(document.Objects[0].UUID) {
		return nil, &Error{Category: "decode_error"}
	}
	return &document.Objects[0], nil
}

// UpdateTag performs one documented PUT and decodes the returned tag object.
func (c *Client) UpdateTag(ctx context.Context, uuid string, input TagUpdateRequest) (*Tag, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	var tag Tag
	if err := c.doTagWrite(ctx, http.MethodPut, "tags/"+uuid+"/", payload, http.StatusOK, &tag); err != nil {
		return nil, err
	}
	if !driveSegment.MatchString(tag.UUID) {
		return nil, &Error{Category: "decode_error"}
	}
	return &tag, nil
}

// DeleteTag sends the documented DELETE acknowledgement and does not poll.
func (c *Client) DeleteTag(ctx context.Context, uuid string) error {
	if !driveSegment.MatchString(uuid) {
		return &Error{Category: "transport"}
	}
	return c.doTagWrite(ctx, http.MethodDelete, "tags/"+uuid+"/", nil, http.StatusNoContent, nil)
}

func (c *Client) doTagWrite(ctx context.Context, method, route string, payload []byte, wantStatus int, out any) error {
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	if err := ctx.Err(); err != nil {
		return remoteContextError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+route, bytes.NewReader(payload))
	if err != nil {
		return &Error{Category: "transport"}
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
		return remoteContextError(ctx, err)
	}
	if resp == nil {
		return &Error{Category: "transport"}
	}
	fail := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != wantStatus {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return fail(category)
	}
	if enc := resp.Header.Get("Content-Encoding"); enc != "" && !strings.EqualFold(enc, "identity") {
		return fail("unsupported_encoding")
	}
	if out == nil {
		if err := ctx.Err(); err != nil {
			return remoteContextError(ctx, err)
		}
		return nil
	}
	if resp.Body == nil {
		return fail("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return remoteContextError(ctx, ctxErr)
		}
		return fail("transport")
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return remoteContextError(ctx, ctxErr)
	}
	if len(body) > responseLimit {
		return fail("response_too_large")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(out); err != nil {
		return fail("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fail("decode_error")
	}
	return nil
}
func decodeTags(b []byte) ([]Tag, error) {
	var root struct {
		Objects []Tag `json:"objects"`
		Meta    *struct {
			Limit      int `json:"limit"`
			Offset     int `json:"offset"`
			TotalCount int `json:"total_count"`
		} `json:"meta"`
	}
	e := json.NewDecoder(bytes.NewReader(b)).Decode(&root)
	return root.Objects, e
}

func decodeTag(b []byte) (*Tag, error) {
	var tag Tag
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&tag); err != nil {
		return nil, err
	}
	return &tag, nil
}
