package cloudsigmaapi

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultSnapshotWaitTimeout = 10 * time.Minute
	defaultSnapshotWaitDelay   = 3 * time.Second
)

// SnapshotStatusError reports a snapshot status that the existing provider
// waiter cannot reconcile to a successful available state.
type SnapshotStatusError struct {
	UUID   string
	Status string
}

func (e *SnapshotStatusError) Error() string {
	return fmt.Sprintf("snapshot %s reached unexpected status %q", e.UUID, e.Status)
}

// WaitSnapshotAvailable polls the native snapshot read path using the same
// pending and target states as the provider's legacy waiter. Caller
// cancellation and deadlines take precedence over the default timeout.
func (c *Client) WaitSnapshotAvailable(ctx context.Context, uuid string, timeout, delay time.Duration) (*Snapshot, error) {
	if timeout <= 0 {
		timeout = defaultSnapshotWaitTimeout
	}
	if delay <= 0 {
		delay = defaultSnapshotWaitDelay
	}
	// Keep the caller's request deadline policy intact. The legacy waiter owns
	// its overall timeout outside the refresh request, so native reads must not
	// acquire a synthetic deadline when the caller supplied none.
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()
	timedOut := make(chan struct{})
	go func() {
		select {
		case <-timeoutTimer.C:
			close(timedOut)
			cancel()
		case <-waitCtx.Done():
		}
	}()

	for {
		if err := waitCtx.Err(); err != nil {
			select {
			case <-timedOut:
				return nil, &Error{Category: "timeout"}
			default:
			}
			return nil, remoteContextError(waitCtx, err)
		}
		snapshot, err := c.SnapshotWithCallerContext(waitCtx, uuid)
		if err != nil {
			return nil, err
		}
		switch snapshot.Status {
		case "available":
			return snapshot, nil
		case "creating", "cloning_dst":
			// Continue polling pending states accepted by the existing waiter.
		default:
			return nil, &SnapshotStatusError{UUID: uuid, Status: snapshot.Status}
		}

		timer := time.NewTimer(delay)
		select {
		case <-waitCtx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			select {
			case <-timedOut:
				return nil, &Error{Category: "timeout"}
			default:
			}
			return nil, remoteContextError(waitCtx, waitCtx.Err())
		case <-timer.C:
		}
	}
}
