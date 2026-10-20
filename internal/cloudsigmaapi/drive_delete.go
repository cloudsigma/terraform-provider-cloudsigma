package cloudsigmaapi

import (
	"context"
	"errors"
	"net/http"
)

// DeleteDrive deletes a drive without polling for disappearance.
// It never retries an outcome whose commit status is unknown.
func (c *Client) DeleteDrive(ctx context.Context, uuid string) error {
	if !driveSegment.MatchString(uuid) {
		return &Error{Category: "transport"}
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
		return contextFailure(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.base+"drives/"+uuid+"/", nil)
	if err != nil {
		return &Error{Category: "transport"}
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
		return contextFailure(err)
	}
	if err := ctx.Err(); err != nil {
		return contextFailure(err)
	}
	if resp == nil {
		return &Error{Category: "transport"}
	}
	if resp.StatusCode != http.StatusNoContent {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	return nil
}
