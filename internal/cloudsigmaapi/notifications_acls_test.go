package cloudsigmaapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestNotificationACLNativeReads(t *testing.T) {
	for _, tc := range []struct {
		name, route string
		call        func(context.Context, *Client) (DynamicDocument, error)
	}{
		{name: "contacts", route: "notification_contacts", call: func(ctx context.Context, c *Client) (DynamicDocument, error) { return c.NotificationContacts(ctx) }},
		{name: "preferences", route: "notification_preferences", call: func(ctx context.Context, c *Client) (DynamicDocument, error) { return c.NotificationPreferences(ctx) }},
		{name: "acls", route: "acls", call: func(ctx context.Context, c *Client) (DynamicDocument, error) { return c.ACLs(ctx) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/"+tc.route+"/" || req.URL.RawQuery != "limit=0" {
					t.Fatalf("request=%s %s", req.Method, req.URL.String())
				}
				if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
					t.Fatal("missing basic auth")
				}
				return remoteSnapshotResponse(http.StatusOK, `{"meta":{"limit":0,"offset":0,"total_count":1},"objects":[{"uuid":"object-1","unknown":{"nested":true}}]}`), nil
			})
			got, err := tc.call(context.Background(), c)
			if err != nil || string(got["objects"]) != `[{"uuid":"object-1","unknown":{"nested":true}}]` {
				t.Fatalf("document=%s err=%v", got["objects"], err)
			}
		})
	}
}

func TestNotificationACLReadBearerAndFailures(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatal("missing bearer auth")
		}
		return remoteSnapshotResponse(http.StatusOK, `{"meta":{"total_count":0},"objects":[]}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if _, err := c.NotificationPreferences(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		status     int
		body, want string
	}{
		{status: http.StatusFound, body: "private", want: "http_status"},
		{status: http.StatusAccepted, body: "private", want: "unexpected_status"},
		{status: http.StatusOK, body: `{`, want: "decode_error"},
		{status: http.StatusOK, body: `{"objects":[]} {}`, want: "decode_error"},
		{status: http.StatusOK, body: `null`, want: "decode_error"},
	} {
		c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return remoteSnapshotResponse(tc.status, tc.body), nil })
		_, err := c.ACLs(context.Background())
		if err == nil || !strings.HasPrefix(err.Error(), tc.want) || strings.Contains(err.Error(), "private") {
			t.Fatalf("err=%v want=%s", err, tc.want)
		}
	}
}
