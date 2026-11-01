package cloudsigmaapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWaitSnapshotAvailableImmediateAndProgression(t *testing.T) {
	calls := 0
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		calls++
		body := `{"uuid":"snapshot-1","drive":{"uuid":"drive-1"},"status":"available"}`
		if calls == 1 {
			body = `{"uuid":"snapshot-1","drive":{"uuid":"drive-1"},"status":"creating"}`
		}
		return remoteSnapshotResponse(http.StatusOK, body), nil
	})
	got, err := c.WaitSnapshotAvailable(context.Background(), "snapshot-1", time.Second, time.Millisecond)
	if err != nil || got == nil || got.Status != "available" || calls != 2 {
		t.Fatalf("snapshot=%+v err=%v calls=%d", got, err, calls)
	}

	calls = 0
	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
		calls++
		return remoteSnapshotResponse(http.StatusOK, `{"uuid":"snapshot-1","drive":{"uuid":"drive-1"},"status":"available"}`), nil
	})
	got, err = c.WaitSnapshotAvailable(context.Background(), "snapshot-1", time.Second, time.Millisecond)
	if err != nil || got == nil || calls != 1 {
		t.Fatalf("immediate snapshot=%+v err=%v calls=%d", got, err, calls)
	}
}

func TestWaitSnapshotAvailableRejectsTerminalAndUnknownStatuses(t *testing.T) {
	for _, status := range []string{"unavailable", "transferring", ""} {
		c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
			return remoteSnapshotResponse(http.StatusOK, `{"uuid":"snapshot-1","drive":{"uuid":"drive-1"},"status":"`+status+`"}`), nil
		})
		_, err := c.WaitSnapshotAvailable(context.Background(), "snapshot-1", time.Second, time.Millisecond)
		var statusErr *SnapshotStatusError
		if !errors.As(err, &statusErr) || statusErr.Status != status {
			t.Fatalf("status=%q err=%v", status, err)
		}
	}
}

func TestWaitSnapshotAvailableRejectsInvalidAndPropagatesFailures(t *testing.T) {
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid UUID dispatched")
		return nil, nil
	})
	if _, err := c.WaitSnapshotAvailable(context.Background(), "../bad", time.Second, time.Millisecond); err == nil {
		t.Fatal("invalid UUID accepted")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) { t.Fatal("cancel dispatched"); return nil, nil })
	if _, err := c.WaitSnapshotAvailable(ctx, "snapshot-1", time.Second, time.Millisecond); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("cancel err=%v", err)
	}

	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("private transport") })
	if _, err := c.WaitSnapshotAvailable(context.Background(), "snapshot-1", time.Second, time.Millisecond); err == nil || err.Error() != "transport" {
		t.Fatalf("transport err=%v", err)
	}
}

func TestWaitSnapshotAvailableHonorsTimeoutWhilePending(t *testing.T) {
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
		return remoteSnapshotResponse(http.StatusOK, `{"uuid":"snapshot-1","drive":{"uuid":"drive-1"},"status":"creating"}`), nil
	})
	_, err := c.WaitSnapshotAvailable(context.Background(), "snapshot-1", 2*time.Millisecond, time.Second)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("timeout err=%v", err)
	}
}
