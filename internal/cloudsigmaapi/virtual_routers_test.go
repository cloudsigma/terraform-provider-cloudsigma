package cloudsigmaapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestVirtualRoutersReadsDynamicCollection(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/virtual_routers/" || req.URL.RawQuery != "limit=0" {
			t.Fatalf("request=%s %s", req.Method, req.URL.String())
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
			t.Fatalf("auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{"limit":0,"offset":0,"total_count":1},"objects":[{"uuid":"router-1","configuration":{"name":"edge"},"state":"started"}]}`), nil
	})
	got, err := c.VirtualRouters(context.Background())
	if err != nil || string(got["meta"]) != `{"limit":0,"offset":0,"total_count":1}` {
		t.Fatalf("routers=%s err=%v", got["meta"], err)
	}
}

func TestVirtualRoutersBearerAndFailures(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"objects":[]}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if _, err := c.VirtualRouters(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{name: "redirect", status: http.StatusFound, want: "http_status"},
		{name: "unexpected", status: http.StatusAccepted, want: "unexpected_status"},
		{name: "malformed", status: http.StatusOK, body: `{`, want: "decode_error"},
		{name: "trailing", status: http.StatusOK, body: `{"objects":[]} {}`, want: "decode_error"},
		{name: "null", status: http.StatusOK, body: `null`, want: "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return remoteSnapshotResponse(tc.status, tc.body), nil })
			_, err := c.VirtualRouters(context.Background())
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("private transport") })
	if _, err := c.VirtualRouters(context.Background()); err == nil || err.Error() != "transport" {
		t.Fatalf("transport=%v", err)
	}
}
