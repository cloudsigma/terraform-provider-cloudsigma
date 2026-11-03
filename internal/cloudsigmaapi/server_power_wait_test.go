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

func TestStartServerAndWaitRunningStrictActionAndReadReconciliation(t *testing.T) {
	statuses := []string{"starting", "running"}
	calls := 0
	c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if req.Method != http.MethodPost || req.URL.RequestURI() != "/api/2.0/servers/server-1/action/?do=start" {
				t.Fatalf("action request=%s %s", req.Method, req.URL)
			}
			return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"action":"start","result":"success","uuid":"job-1"}`)), Request: req}, nil
		}
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/servers/server-1/" {
			t.Fatalf("read request=%s %s", req.Method, req.URL)
		}
		status := statuses[calls-2]
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"uuid":"server-1","status":%q}`, status))), Request: req}, nil
	})
	got, err := c.StartServerAndWaitRunning(context.Background(), "server-1", time.Second, time.Millisecond)
	if err != nil || got == nil || got.Status != "running" || calls != 3 {
		t.Fatalf("server=%+v err=%v calls=%d", got, err, calls)
	}
}

func TestShutdownServerAndWaitStoppedStrictActionAndReadReconciliation(t *testing.T) {
	statuses := []string{"stopping", "running", "stopped"}
	calls := 0
	c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if req.Method != http.MethodPost || req.URL.RequestURI() != "/api/2.0/servers/server-1/action/?do=shutdown" {
				t.Fatalf("action request=%s %s", req.Method, req.URL)
			}
			return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"action":"shutdown","result":"success","uuid":"job-1"}`)), Request: req}, nil
		}
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/servers/server-1/" {
			t.Fatalf("read request=%s %s", req.Method, req.URL)
		}
		status := statuses[calls-2]
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"uuid":"server-1","status":%q}`, status))), Request: req}, nil
	})
	got, err := c.ShutdownServerAndWaitStopped(context.Background(), "server-1", time.Second, time.Millisecond)
	if err != nil || got == nil || got.Status != "stopped" || calls != 4 {
		t.Fatalf("server=%+v err=%v calls=%d", got, err, calls)
	}
}

func TestShutdownServerAndWaitStoppedDoesNotReplayOnUncertainOutcome(t *testing.T) {
	posts := 0
	c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			posts++
			return nil, errors.New("connection lost")
		}
		t.Fatal("unexpected follow-up request")
		return nil, nil
	})
	if _, err := c.ShutdownServerAndWaitStopped(context.Background(), "server-1", time.Second, time.Millisecond); err == nil || posts != 1 {
		t.Fatalf("err=%v posts=%d", err, posts)
	}
}

func TestStopServerAndWaitStoppedRejectsUnqualifiedActions(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{name: "200", code: http.StatusOK, body: `{"action":"stop","result":"success","uuid":"job"}`},
		{name: "redirect", code: http.StatusTemporaryRedirect, body: `private`},
		{name: "wrong action", code: http.StatusAccepted, body: `{"action":"start","result":"success","uuid":"job"}`},
		{name: "failed result", code: http.StatusAccepted, body: `{"action":"stop","result":"failed","uuid":"job"}`},
		{name: "missing operation uuid", code: http.StatusAccepted, body: `{"action":"stop","result":"success"}`},
		{name: "trailing", code: http.StatusAccepted, body: `{"action":"stop","result":"success","uuid":"job"} {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: tc.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
			})
			_, err := c.StopServerAndWaitStopped(context.Background(), "server-1", time.Second, time.Millisecond)
			if err == nil || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestServerPowerAndWaitContextFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("private") })
	if _, err := c.StartServerAndWaitRunning(ctx, "server-1", time.Second, time.Millisecond); err == nil || err.Error() != "cancelled" || calls != 0 {
		t.Fatalf("pre-cancel err=%v calls=%d", err, calls)
	}
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) { return nil, errors.New("private transport") })
	if _, err := c.StartServerAndWaitRunning(context.Background(), "server-1", time.Second, time.Millisecond); err == nil || err.Error() != "transport" {
		t.Fatalf("transport err=%v", err)
	}
	if _, err := c.StartServerAndWaitRunning(context.Background(), "../bad", time.Second, time.Millisecond); err == nil {
		t.Fatal("invalid uuid accepted")
	}
}
