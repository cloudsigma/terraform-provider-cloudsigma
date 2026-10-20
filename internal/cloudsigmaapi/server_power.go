package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// StartServer acknowledges a start request; it does not establish running state.
func (c *Client) StartServer(ctx context.Context, uuid string) error {
	return c.serverPower(ctx, uuid, "start")
}

// StopServer acknowledges a stop request; it does not establish stopped state.
func (c *Client) StopServer(ctx context.Context, uuid string) error {
	return c.serverPower(ctx, uuid, "stop")
}

func (c *Client) serverPower(ctx context.Context, uuid, action string) error {
	failure := func(err error) error {
		category := "transport"
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			category = "cancelled"
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		return &Error{Category: category}
	}
	if !driveSegment.MatchString(uuid) {
		return failure(nil)
	}
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"servers/"+uuid+"/action/?do="+action, strings.NewReader("{}"))
	if err != nil {
		return failure(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	req.SetBasicAuth(c.username, c.password)
	// Legacy power sends are bounded by their supplied context, not a new timeout.
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	resp, err := transport.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return failure(err)
	}
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	if resp == nil {
		return failure(nil)
	}
	responseFailure := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	// Preserve pinned SDK all-2xx admission, without interpreting result as completion.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseFailure("http_status")
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return responseFailure("unsupported_encoding")
	}
	if resp.Body == nil {
		return responseFailure("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return failure(err)
	}
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	if len(body) > responseLimit {
		return responseFailure("response_too_large")
	}
	var ack struct {
		Action string `json:"action"`
		Result string `json:"result"`
		UUID   string `json:"uuid"`
	}
	if err := json.NewDecoder(strings.NewReader(string(body))).Decode(&ack); err != nil {
		return responseFailure("decode_error")
	}
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	return nil
}
