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

// KeypairUpdateRequest contains only fields emitted by the registered SSH Update callback.
type KeypairUpdateRequest struct {
	Name       string `json:"name,omitempty"`
	UUID       string `json:"uuid,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
}

func (KeypairUpdateRequest) String() string   { return "KeypairUpdateRequest{redacted}" }
func (KeypairUpdateRequest) GoString() string { return "KeypairUpdateRequest{redacted}" }

// UpdateKeypair dispatches once without polling, OTP exchange, or replay.
func (c *Client) UpdateKeypair(ctx context.Context, uuid string, input KeypairUpdateRequest) (*Keypair, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
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
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.base+"keypairs/"+uuid+"/", bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	req.SetBasicAuth(c.username, c.password)
	resp, err := c.transport.Do(req)
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
	if resp.StatusCode != http.StatusOK {
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
	if err := ctx.Err(); err != nil {
		return nil, contextFailure(err)
	}
	if len(body) > responseLimit {
		return failure("response_too_large")
	}
	result, err := decodeKeypair(body)
	if err != nil {
		return failure("decode_error")
	}
	return result, nil
}
