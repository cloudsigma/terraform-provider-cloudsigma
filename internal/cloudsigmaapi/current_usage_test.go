package cloudsigmaapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCurrentUsageReadsDynamicSingleton(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/currentusage/" || req.URL.RawQuery != "" {
			t.Fatalf("request=%s %s", req.Method, req.URL)
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
			t.Fatalf("auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"balance":{"balance":"12.5","currency":"EUR"},"usage":{"amd_cpu":{"burst":1,"subscribed":2,"using":3}}}`), nil
	})
	got, err := c.CurrentUsage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(got["balance"]) != `{"balance":"12.5","currency":"EUR"}` || string(got["usage"]) != `{"amd_cpu":{"burst":1,"subscribed":2,"using":3}}` {
		t.Fatalf("usage=%s", got["usage"])
	}
}

func TestCurrentUsageSupportsBearerAndRejectsInvalidResponses(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"usage":{}}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if _, err := c.CurrentUsage(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{name: "redirect", status: http.StatusFound, want: "http_status"},
		{name: "unexpected", status: http.StatusAccepted, want: "unexpected_status"},
		{name: "malformed", status: http.StatusOK, body: `{`, want: "decode_error"},
		{name: "trailing", status: http.StatusOK, body: `{"usage":{}} {}`, want: "decode_error"},
		{name: "null", status: http.StatusOK, body: `null`, want: "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(tc.status, tc.body), nil
			})
			_, err := c.CurrentUsage(context.Background())
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("private transport") })
	if _, err := c.CurrentUsage(context.Background()); err == nil || err.Error() != "transport" {
		t.Fatalf("transport err=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, context.Canceled })
	if _, err := c.CurrentUsage(ctx); err == nil || err.Error() != "cancelled" {
		t.Fatalf("cancel err=%v", err)
	}
}
