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

// KeypairCreateRequest is the writable subset used by the SSH key resource.
// Response-only fields such as uuid, fingerprint and resource_uri are never
// projected into a create request.
type KeypairCreateRequest struct {
	Name       string `json:"name,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
}

// CreateKeypair performs one Basic-authenticated POST. It accepts the API's
// object envelope and never retries or polls. A validated acknowledgement is
// returned alongside a cancellation error when cancellation races the body
// read, so callers can retain the acknowledged identity safely.
func (c *Client) CreateKeypair(ctx context.Context, input KeypairCreateRequest) (*Keypair, error) {
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
		Objects []KeypairCreateRequest `json:"objects"`
	}{[]KeypairCreateRequest{input}})
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"keypairs/", bytes.NewReader(payload))
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
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return nil, &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	failure := func(category string) (*Keypair, error) {
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
	if len(body) > responseLimit {
		return failure("response_too_large")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var document struct {
		Objects []Keypair `json:"objects"`
	}
	if err := decoder.Decode(&document); err != nil {
		return failure("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return failure("decode_error")
	}
	if len(document.Objects) != 1 || !driveSegment.MatchString(document.Objects[0].UUID) {
		return failure("decode_error")
	}
	result := &document.Objects[0]
	if err := ctx.Err(); err != nil {
		return result, contextFailure(err)
	}
	return result, nil
}
