package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// StartServerAndWaitRunning starts a server using the documented 202 action
// contract, then reconciles terminal running state through native reads.
// It does not replay the action after an uncertain response.
func (c *Client) StartServerAndWaitRunning(ctx context.Context, uuid string, timeout, delay time.Duration) (*Server, error) {
	if err := c.serverPowerStrict(ctx, uuid, "start"); err != nil {
		return nil, err
	}
	return c.WaitServerRunning(ctx, uuid, timeout, delay)
}

// StopServerAndWaitStopped stops a server using the documented 202 action
// contract, then reconciles terminal stopped state through native reads.
// It does not replay the action after an uncertain response.
func (c *Client) StopServerAndWaitStopped(ctx context.Context, uuid string, timeout, delay time.Duration) (*Server, error) {
	if err := c.serverPowerStrict(ctx, uuid, "stop"); err != nil {
		return nil, err
	}
	return c.WaitServerStopped(ctx, uuid, timeout, delay)
}

// ShutdownServerAndWaitStopped sends the documented ACPI shutdown action and
// reconciles the server through native reads until it reports stopped. The
// backend may return to running when the guest does not complete shutdown;
// that outcome remains subject to the bounded waiter timeout.
func (c *Client) ShutdownServerAndWaitStopped(ctx context.Context, uuid string, timeout, delay time.Duration) (*Server, error) {
	if err := c.serverPowerStrict(ctx, uuid, "shutdown"); err != nil {
		return nil, err
	}
	return c.WaitServerStopped(ctx, uuid, timeout, delay)
}

// serverPowerStrict is a separate seam from serverPower so existing SDK
// compatibility behavior remains unchanged. The API's action UUID identifies
// the accepted operation, not the target server UUID.
func (c *Client) serverPowerStrict(ctx context.Context, uuid, action string) error {
	if !driveSegment.MatchString(uuid) || (action != "start" && action != "stop" && action != "shutdown") {
		return &Error{Category: "transport"}
	}
	if err := ctx.Err(); err != nil {
		return remoteContextError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"servers/"+uuid+"/action/?do="+action, strings.NewReader("{}"))
	if err != nil {
		return &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	c.authenticate(req)
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
		return strictPowerContextError(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return strictPowerContextError(ctx, err)
	}
	if resp == nil {
		return &Error{Category: "transport"}
	}
	fail := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusAccepted {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return fail(category)
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return fail("unsupported_encoding")
	}
	if resp.Body == nil {
		return fail("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return strictPowerContextError(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return strictPowerContextError(ctx, err)
	}
	if len(body) > responseLimit {
		return fail("response_too_large")
	}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	var ack struct {
		Action string `json:"action"`
		Result string `json:"result"`
		UUID   string `json:"uuid"`
	}
	if err := dec.Decode(&ack); err != nil {
		return fail("decode_error")
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return fail("decode_error")
	}
	if ack.Action != action || ack.Result != "success" || !driveSegment.MatchString(ack.UUID) {
		return fail("decode_error")
	}
	return nil
}

func strictPowerContextError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return &Error{Category: "cancelled"}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return &Error{Category: "timeout"}
	}
	return &Error{Category: "transport"}
}
