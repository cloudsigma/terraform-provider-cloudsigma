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

// AssignDriveScheduler associates one backup scheduler with a drive. The API
// acknowledges this action with 202; the returned drive is only an
// acknowledgement and does not imply that a backup has completed.
func (c *Client) AssignDriveScheduler(ctx context.Context, driveUUID, schedulerUUID string) (*Drive, error) {
	return c.driveSchedulerAction(ctx, driveUUID, schedulerUUID, "set_scheduler")
}

// UnassignDriveScheduler removes the drive's backup scheduler association. The
// API uses the same scheduler UUID field as assignment and acknowledges the
// action with 202. It does not imply any backup or deletion completion.
func (c *Client) UnassignDriveScheduler(ctx context.Context, driveUUID, schedulerUUID string) (*Drive, error) {
	return c.driveSchedulerAction(ctx, driveUUID, schedulerUUID, "remove_scheduler")
}

func (c *Client) driveSchedulerAction(ctx context.Context, driveUUID, schedulerUUID, action string) (*Drive, error) {
	if !driveSegment.MatchString(driveUUID) || !driveSegment.MatchString(schedulerUUID) {
		return nil, &Error{Category: "transport"}
	}
	if action != "set_scheduler" && action != "remove_scheduler" {
		return nil, &Error{Category: "transport"}
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, driveSchedulerContextError(ctx, err)
	}

	payload, err := json.Marshal(struct {
		BackupPolicyUUID string `json:"backup_policy_uuid"`
	}{BackupPolicyUUID: schedulerUUID})
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.base+"drives/"+driveUUID+"/action/?do="+action, bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Category: "transport"}
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
		return nil, driveSchedulerContextError(ctx, err)
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	failure := func(category string) (*Drive, error) {
		return nil, &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusAccepted {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return failure(category)
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return failure("unsupported_encoding")
	}
	if resp.Body == nil {
		return failure("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return nil, driveSchedulerContextError(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, driveSchedulerContextError(ctx, err)
	}
	if len(body) > responseLimit {
		return failure("response_too_large")
	}
	var drive Drive
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&drive); err != nil || drive.UUID != driveUUID {
		return failure("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return failure("decode_error")
	}
	if err := ctx.Err(); err != nil {
		return nil, driveSchedulerContextError(ctx, err)
	}
	return &drive, nil
}

func driveSchedulerContextError(ctx context.Context, err error) error {
	category := "transport"
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		category = "cancelled"
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		category = "timeout"
	}
	return &Error{Category: category}
}
