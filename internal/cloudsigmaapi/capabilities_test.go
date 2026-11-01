package cloudsigmaapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestCapabilitiesReadsDynamicDocument(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.RequestURI() != "/api/2.0/capabilities/?limit=0" {
			t.Fatalf("route=%s %s", req.Method, req.URL.RequestURI())
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"hosts":{"amd":{"cpu":{"min":1000}}},"remote_snapshots":{"remote_locations":["gva"]}}`), nil
	})
	got, err := c.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(got["hosts"]) != `{"amd":{"cpu":{"min":1000}}}` || string(got["remote_snapshots"]) != `{"remote_locations":["gva"]}` {
		t.Fatalf("capabilities=%s", got)
	}
}

func TestCapabilitiesUsesBearerAndRejectsInvalidResponses(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"snapshots":{"max":5}}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if got, err := c.Capabilities(context.Background()); err != nil || string(got["snapshots"]) != `{"max":5}` {
		t.Fatalf("capabilities=%s err=%v", got, err)
	}

	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "redirect", status: http.StatusFound},
		{name: "malformed", status: http.StatusOK, body: `{"hosts":`},
		{name: "null", status: http.StatusOK, body: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(tc.status, tc.body), nil
			})
			_, err := c.Capabilities(context.Background())
			if err == nil {
				t.Fatal("invalid capability response accepted")
			}
		})
	}

	c = remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("unknown outcome") })
	if _, err := c.Capabilities(context.Background()); err == nil {
		t.Fatal("transport error accepted")
	}
}
