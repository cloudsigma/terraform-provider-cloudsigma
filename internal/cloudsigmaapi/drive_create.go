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

// DriveCreateRequest is the writable nonclone subset used by the provider callback.
// Response-only fields cannot be echoed into a create request.
type DriveCreateRequest struct {
	Media       string `json:"media,omitempty"`
	Name        string `json:"name,omitempty"`
	Size        int    `json:"size,omitempty"`
	StorageType string `json:"storage_type,omitempty"`
}

// CreateDrive performs one POST using only the caller's deadline; it never polls or retries.
func (c *Client) CreateDrive(ctx context.Context, input DriveCreateRequest) (*Drive, error) {
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
		return nil, contextFailure(err)
	}
	payload, err := json.Marshal(struct {
		Objects []DriveCreateRequest `json:"objects"`
	}{[]DriveCreateRequest{input}})
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"drives/", bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Category: "transport"}
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
		return nil, contextFailure(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, contextFailure(err)
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	if resp.StatusCode != http.StatusCreated {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return nil, &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	failure := func(category string) (*Drive, error) {
		return nil, &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return failure("unsupported_encoding")
	}
	if resp.Body == nil {
		return failure("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return nil, contextFailure(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, contextFailure(err)
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
	result, err := decodeDrives(document)
	if err != nil || len(result) == 0 || !driveSegment.MatchString(result[0].UUID) {
		return failure("decode_error")
	}
	return &result[0], nil
}
