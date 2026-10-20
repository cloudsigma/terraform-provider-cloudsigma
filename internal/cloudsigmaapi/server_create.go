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

// ServerCreateRequest is the existing provider callback writable subset.
// Nested references retain their SDK-compatible wire representation.
type ServerCreateRequest struct {
	CPU               int                    `json:"cpu,omitempty"`
	EnclavePageCaches []EnclavePageCache     `json:"epcs,omitempty"`
	Memory            int                    `json:"mem,omitempty"`
	Meta              map[string]interface{} `json:"meta,omitempty"`
	Name              string                 `json:"name,omitempty"`
	NICs              []ServerNIC            `json:"nics,omitempty"`
	PublicKeys        []Keypair              `json:"pubkeys,omitempty"`
	SMP               int                    `json:"smp,omitempty"`
	Tags              []Tag                  `json:"tags,omitempty"`
	VNCPassword       string                 `json:"vnc_password,omitempty"`
}

// CreateServer performs one POST, preserving a fully validated acknowledgment
// across a cancellation race. It never retries, polls, or performs a fallback.
func (c *Client) CreateServer(ctx context.Context, input ServerCreateRequest) (*Server, error) {
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
		Objects []ServerCreateRequest `json:"objects"`
	}{[]ServerCreateRequest{input}})
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"servers/", bytes.NewReader(payload))
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
	if resp.StatusCode != http.StatusCreated {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return nil, &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	failure := func(category string) (*Server, error) {
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
	var root struct {
		Objects []serverResponse `json:"objects"`
	}
	err = json.Unmarshal(document, &root)
	result := root.Objects
	if err != nil || len(result) != 1 || !driveSegment.MatchString(result[0].UUID) {
		return failure("decode_error")
	}
	// A complete validated acknowledgment remains useful for reconciliation even
	// when cancellation races the response. Never infer identity from failed reads.
	if err := ctx.Err(); err != nil {
		return (*Server)(&result[0]), contextFailure(err)
	}
	return (*Server)(&result[0]), nil
}
