package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type remoteSnapshotDoer func(*http.Request) (*http.Response, error)

func (f remoteSnapshotDoer) Do(req *http.Request) (*http.Response, error) { return f(req) }

func remoteSnapshotClient(do remoteSnapshotDoer) *Client {
	return &Client{
		transport: do,
		base:      "https://fixture.example/api/2.0/",
		username:  "fixture-user",
		password:  "fixture-password",
		userAgent: "fixture-agent",
	}
}

func remoteSnapshotResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestMarshalRemoteSnapshotRequestProjectsWritableFields(t *testing.T) {
	tests := []struct {
		name  string
		input []RemoteSnapshotCreate
		batch bool
		check func(t *testing.T, body map[string]any)
	}{
		{
			name:  "bare uuid with default location omitted",
			input: []RemoteSnapshotCreate{{Drive: "drive-1"}},
			check: func(t *testing.T, body map[string]any) {
				if got, ok := body["drive"].(string); !ok || got != "drive-1" {
					t.Fatalf("drive=%#v", body["drive"])
				}
				if _, ok := body["location"]; ok {
					t.Fatalf("default location must be omitted: %#v", body)
				}
			},
		},
		{
			name:  "bare minimal nested drive reference",
			input: []RemoteSnapshotCreate{{Drive: RemoteSnapshotDriveRef{UUID: "drive-2"}, Location: "far"}},
			check: func(t *testing.T, body map[string]any) {
				got, ok := body["drive"].(map[string]any)
				if !ok || len(got) != 1 || got["uuid"] != "drive-2" {
					t.Fatalf("drive=%#v", body["drive"])
				}
				if body["location"] != "far" {
					t.Fatalf("location=%#v", body["location"])
				}
			},
		},
		{
			name: "batch preserves projection and order",
			input: []RemoteSnapshotCreate{
				{Drive: "drive-a", Location: "zrh"},
				{Drive: &RemoteSnapshotDriveRef{UUID: "drive-b"}},
			},
			batch: true,
			check: func(t *testing.T, body map[string]any) {
				objects, ok := body["objects"].([]any)
				if !ok || len(objects) != 2 {
					t.Fatalf("objects=%#v", body["objects"])
				}
				first := objects[0].(map[string]any)
				second := objects[1].(map[string]any)
				if first["drive"] != "drive-a" || first["location"] != "zrh" {
					t.Fatalf("first=%#v", first)
				}
				if second["drive"].(map[string]any)["uuid"] != "drive-b" || len(second) != 1 {
					t.Fatalf("second=%#v", second)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := marshalRemoteSnapshotRequest(tc.input, tc.batch)
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal(payload, &body); err != nil {
				t.Fatal(err)
			}
			if !tc.batch && strings.Contains(string(payload), "objects") {
				t.Fatalf("bare request unexpectedly wrapped: %s", payload)
			}
			tc.check(t, body)
		})
	}

	if _, err := marshalRemoteSnapshotRequest([]RemoteSnapshotCreate{{Drive: Drive{UUID: "drive"}}}, true); err == nil {
		t.Fatal("complete response drive object must be rejected")
	}
	if _, err := marshalRemoteSnapshotRequest([]RemoteSnapshotCreate{{Drive: "drive", Location: "not an alias"}}, true); err == nil {
		t.Fatal("invalid location label must be rejected")
	}
}

func TestCreateRemoteSnapshotsOrderStatusAndDriveMeta(t *testing.T) {
	var requestBody []byte
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/2.0/remotesnapshots/" {
			return nil, errors.New("unexpected route")
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
			return nil, errors.New("unexpected authentication")
		}
		requestBody, _ = io.ReadAll(req.Body)
		return remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"result-a","status":"creating","drive_meta":{"uuid":"response-only-a"}},{"uuid":"result-b","status":"transferring","drive_meta":{"uuid":"response-only-b"}}]}`), nil
	})
	got, err := c.CreateRemoteSnapshots(context.Background(), []RemoteSnapshotCreate{{Drive: "drive-a"}, {Drive: RemoteSnapshotDriveRef{UUID: "drive-b"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].UUID != "result-a" || got[1].UUID != "result-b" || got[0].Status != "creating" || got[1].Status != "transferring" {
		t.Fatalf("objects=%+v", got)
	}
	if got[0].Drive != nil || got[1].Drive != nil {
		t.Fatalf("response-only drive_meta leaked into state: %+v", got)
	}
	var body struct {
		Objects []remoteSnapshotCreateWire `json:"objects"`
	}
	if err := json.Unmarshal(requestBody, &body); err != nil || len(body.Objects) != 2 {
		t.Fatalf("request=%s err=%v", requestBody, err)
	}
}

func TestCreateRemoteSnapshotUsesBearerAuthentication(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", got)
		}
		if _, _, ok := req.BasicAuth(); ok {
			t.Fatal("bearer request unexpectedly carried basic credentials")
		}
		return remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"snapshot-token"}]}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	got, err := c.CreateRemoteSnapshot(context.Background(), RemoteSnapshotCreate{Drive: "drive-token"})
	if err != nil || got.UUID != "snapshot-token" {
		t.Fatalf("snapshot=%+v err=%v", got, err)
	}
}

func TestCreateRemoteSnapshotFailureClassification(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		err      error
		want     string
	}{
		{name: "redirect refused", response: remoteSnapshotResponse(http.StatusFound, ""), want: "http_status"},
		{name: "unknown success status", response: remoteSnapshotResponse(http.StatusAccepted, ""), want: "unexpected_status"},
		{name: "malformed response", response: remoteSnapshotResponse(http.StatusCreated, `{"objects":[`), want: "decode_error"},
		{name: "wrong object count", response: remoteSnapshotResponse(http.StatusCreated, `{"objects":[]}`), want: "decode_error"},
		{name: "invalid object uuid", response: remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":""}]}`), want: "decode_error"},
		{name: "unsupported encoding", response: func() *http.Response {
			r := remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"ok"}]}`)
			r.Header.Set("Content-Encoding", "gzip")
			return r
		}(), want: "unsupported_encoding"},
		{name: "transport unknown outcome", err: errors.New("connection outcome unknown"), want: "transport"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return tc.response, tc.err })
			_, err := c.CreateRemoteSnapshot(context.Background(), RemoteSnapshotCreate{Drive: "drive"})
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Category != tc.want {
				t.Fatalf("err=%v want category %q", err, tc.want)
			}
		})
	}
}

func TestCreateRemoteSnapshotContextClassification(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() context.Context
		want string
	}{
		{name: "cancelled before request", ctx: func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }, want: "cancelled"},
		{name: "deadline before request", ctx: func() context.Context {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			return ctx
		}, want: "timeout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return nil, req.Context().Err()
			})
			_, err := c.CreateRemoteSnapshot(tc.ctx(), RemoteSnapshotCreate{Drive: "drive"})
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Category != tc.want || calls != 0 {
				t.Fatalf("err=%v calls=%d want category=%q", err, calls, tc.want)
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	done := make(chan error, 1)
	go func() {
		_, err := c.CreateRemoteSnapshot(ctx, RemoteSnapshotCreate{Drive: "drive"})
		done <- err
	}()
	cancel()
	var apiErr *Error
	select {
	case err := <-done:
		if !errors.As(err, &apiErr) || apiErr.Category != "cancelled" {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not observe cancellation")
	}
}
