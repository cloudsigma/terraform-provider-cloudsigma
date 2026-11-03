package cloudsigmaapi

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultLifecycleWaitTimeout = 10 * time.Minute
	defaultLifecycleWaitDelay   = 3 * time.Second
)

// DriveStatusError reports a drive status that cannot be reconciled to a
// mounted or unmounted terminal state by the native lifecycle waiter.
type DriveStatusError struct {
	UUID   string
	Status string
}

func (e *DriveStatusError) Error() string {
	return fmt.Sprintf("drive %s reached unexpected status %q", e.UUID, e.Status)
}

// ServerStatusError reports a server status that cannot be reconciled to the
// requested running or stopped terminal state by the native lifecycle waiter.
type ServerStatusError struct {
	UUID   string
	Status string
}

func (e *ServerStatusError) Error() string {
	return fmt.Sprintf("server %s reached unexpected status %q", e.UUID, e.Status)
}

// WaitDriveMountedOrUnmounted polls the native drive read path until the drive
// reaches mounted or unmounted. It never retries a failed read and never
// infers completion from an update or action response.
func (c *Client) WaitDriveMountedOrUnmounted(ctx context.Context, uuid string, timeout, delay time.Duration) (*Drive, error) {
	if timeout <= 0 {
		timeout = defaultLifecycleWaitTimeout
	}
	if delay <= 0 {
		delay = defaultLifecycleWaitDelay
	}
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	timedOut := make(chan struct{})
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	go func() {
		select {
		case <-timer.C:
			close(timedOut)
			cancel()
		case <-waitCtx.Done():
		}
	}()

	for {
		if err := waitCtx.Err(); err != nil {
			return nil, lifecycleContextError(waitCtx, timedOut, err)
		}
		drive, err := c.DriveWithCallerContext(waitCtx, uuid)
		if err != nil {
			return nil, err
		}
		if drive == nil || drive.UUID != uuid {
			return nil, &Error{Category: "decode_error"}
		}
		switch drive.Status {
		case "mounted", "unmounted":
			return drive, nil
		case "cloning_dst", "creating", "resizing":
			// Continue polling documented transitional states.
		default:
			return nil, &DriveStatusError{UUID: uuid, Status: drive.Status}
		}
		if err := lifecycleWait(waitCtx, timedOut, delay); err != nil {
			return nil, err
		}
	}
}

// WaitServerRunning polls until a server reports running. It is a read-only
// reconciliation helper; callers must issue start separately.
func (c *Client) WaitServerRunning(ctx context.Context, uuid string, timeout, delay time.Duration) (*Server, error) {
	return c.waitServerStatus(ctx, uuid, "running", timeout, delay)
}

// WaitServerStopped polls until a server reports stopped. It is a read-only
// reconciliation helper; callers must issue stop separately.
func (c *Client) WaitServerStopped(ctx context.Context, uuid string, timeout, delay time.Duration) (*Server, error) {
	return c.waitServerStatus(ctx, uuid, "stopped", timeout, delay)
}

func (c *Client) waitServerStatus(ctx context.Context, uuid, target string, timeout, delay time.Duration) (*Server, error) {
	if target != "running" && target != "stopped" {
		return nil, &Error{Category: "transport"}
	}
	if timeout <= 0 {
		timeout = defaultLifecycleWaitTimeout
	}
	if delay <= 0 {
		delay = defaultLifecycleWaitDelay
	}
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	timedOut := make(chan struct{})
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	go func() {
		select {
		case <-timer.C:
			close(timedOut)
			cancel()
		case <-waitCtx.Done():
		}
	}()

	for {
		if err := waitCtx.Err(); err != nil {
			return nil, lifecycleContextError(waitCtx, timedOut, err)
		}
		server, err := c.ServerWithCallerContext(waitCtx, uuid)
		if err != nil {
			return nil, err
		}
		if server == nil || server.UUID != uuid {
			return nil, &Error{Category: "decode_error"}
		}
		if server.Status == target {
			return server, nil
		}
		// Preserve the existing provider policy for transitional and
		// unavailable power states. Unknown states fail closed.
		if (target == "running" && (server.Status == "stopped" || server.Status == "starting" || server.Status == "unavailable")) ||
			(target == "stopped" && (server.Status == "running" || server.Status == "stopping" || server.Status == "unavailable")) {
			if err := lifecycleWait(waitCtx, timedOut, delay); err != nil {
				return nil, err
			}
			continue
		}
		return nil, &ServerStatusError{UUID: uuid, Status: server.Status}
	}
}

func lifecycleWait(ctx context.Context, timedOut <-chan struct{}, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return lifecycleContextError(ctx, timedOut, ctx.Err())
	case <-timer.C:
		return nil
	}
}

func lifecycleContextError(ctx context.Context, timedOut <-chan struct{}, err error) error {
	select {
	case <-timedOut:
		return &Error{Category: "timeout"}
	default:
		return remoteContextError(ctx, err)
	}
}
