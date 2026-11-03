package cloudsigmaapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWaitDriveMountedOrUnmountedProgressionAndTerminalErrors(t *testing.T) {
	statuses := []string{"creating", "resizing", "unmounted"}
	calls := 0
	c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path != "/api/2.0/drives/drive-1/" {
			t.Fatalf("path=%s", req.URL.Path)
		}
		status := statuses[len(statuses)-1]
		if calls <= len(statuses) {
			status = statuses[calls-1]
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"uuid":"drive-1","status":%q}`, status)))}, nil
	})
	got, err := c.WaitDriveMountedOrUnmounted(context.Background(), "drive-1", time.Second, time.Millisecond)
	if err != nil || got == nil || got.Status != "unmounted" || calls != 3 {
		t.Fatalf("drive=%+v err=%v calls=%d", got, err, calls)
	}

	c = reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"drive-1","status":"unavailable"}`))}, nil
	})
	_, err = c.WaitDriveMountedOrUnmounted(context.Background(), "drive-1", time.Second, time.Millisecond)
	var statusErr *DriveStatusError
	if !errors.As(err, &statusErr) || statusErr.Status != "unavailable" {
		t.Fatalf("unavailable err=%v", err)
	}
}

func TestWaitDriveMountedOrUnmountedCancellationTimeoutAndReadFailure(t *testing.T) {
	calls := 0
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"drive-1","status":"creating"}`))}, nil
	})
	_, err := c.WaitDriveMountedOrUnmounted(context.Background(), "drive-1", 2*time.Millisecond, time.Second)
	if err == nil || err.Error() != "timeout" {
		t.Fatalf("timeout err=%v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("request after cancellation")
		return nil, nil
	})
	_, err = c.WaitDriveMountedOrUnmounted(ctx, "drive-1", time.Second, time.Millisecond)
	if err == nil || err.Error() != "cancelled" {
		t.Fatalf("cancel err=%v", err)
	}

	c = reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("private transport")
	})
	_, err = c.WaitDriveMountedOrUnmounted(context.Background(), "drive-1", time.Second, time.Millisecond)
	if err == nil || err.Error() != "transport" {
		t.Fatalf("read failure err=%v", err)
	}

	if _, err = c.WaitDriveMountedOrUnmounted(context.Background(), "../bad", time.Second, time.Millisecond); err == nil {
		t.Fatal("invalid uuid accepted")
	}
}

func TestWaitServerRunningAndStoppedProgression(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target string
		states []string
		wait   func(*Client, context.Context, string, time.Duration, time.Duration) (*Server, error)
	}{
		{name: "running", target: "running", states: []string{"stopped", "starting", "running"}, wait: (*Client).WaitServerRunning},
		{name: "stopped", target: "stopped", states: []string{"running", "stopping", "stopped"}, wait: (*Client).WaitServerStopped},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/api/2.0/servers/server-1/" {
					t.Fatalf("path=%s", req.URL.Path)
				}
				status := tc.states[len(tc.states)-1]
				if calls <= len(tc.states) {
					status = tc.states[calls-1]
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"uuid":"server-1","status":%q}`, status)))}, nil
			})
			got, err := tc.wait(c, context.Background(), "server-1", time.Second, time.Millisecond)
			if err != nil || got == nil || got.Status != tc.target || calls != len(tc.states) {
				t.Fatalf("server=%+v err=%v calls=%d", got, err, calls)
			}
		})
	}
}

func TestWaitServerStatusRejectsUnknownAndHonorsUnavailablePolicy(t *testing.T) {
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"server-1","status":"failed"}`))}, nil
	})
	_, err := c.WaitServerRunning(context.Background(), "server-1", time.Second, time.Millisecond)
	var statusErr *ServerStatusError
	if !errors.As(err, &statusErr) || statusErr.Status != "failed" {
		t.Fatalf("unknown err=%v", err)
	}

	calls := 0
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) {
		calls++
		status := "unavailable"
		if calls > 1 {
			status = "running"
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"uuid":"server-1","status":%q}`, status)))}, nil
	})
	got, err := c.WaitServerRunning(context.Background(), "server-1", time.Second, time.Millisecond)
	if err != nil || got == nil || got.Status != "running" || calls != 2 {
		t.Fatalf("unavailable progression server=%+v err=%v calls=%d", got, err, calls)
	}
}
