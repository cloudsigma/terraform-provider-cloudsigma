package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDriveSchedulerActionsUseCanonicalRouteBodyAndAuth(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action string
		bearer bool
		call   func(context.Context, *Client) (*Drive, error)
	}{
		{name: "assign basic", action: "set_scheduler", call: func(ctx context.Context, c *Client) (*Drive, error) {
			return c.AssignDriveScheduler(ctx, "drive-1", "scheduler-1")
		}},
		{name: "unassign bearer", action: "remove_scheduler", bearer: true, call: func(ctx context.Context, c *Client) (*Drive, error) {
			return c.UnassignDriveScheduler(ctx, "drive-1", "scheduler-1")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.RequestURI() != "/api/2.0/drives/drive-1/action/?do="+tc.action {
					t.Fatalf("route=%s %s", req.Method, req.URL.RequestURI())
				}
				if tc.bearer {
					if got := req.Header.Get("Authorization"); got != "Bearer fixture-token" {
						t.Fatalf("authorization=%q", got)
					}
				} else {
					user, password, ok := req.BasicAuth()
					if !ok || user != "fixture-user" || password != "fixture-password" {
						t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
					}
				}
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 1 || body["backup_policy_uuid"] != "scheduler-1" {
					t.Fatalf("body=%#v", body)
				}
				return remoteSnapshotResponse(http.StatusAccepted, `{"uuid":"drive-1","backup_scheduler":{"uuid":"scheduler-1"}}`), nil
			})
			if tc.bearer {
				c.username, c.password, c.token = "", "", "fixture-token"
			}
			got, err := tc.call(context.Background(), c)
			if err != nil || got == nil || got.UUID != "drive-1" || got.BackupScheduler == nil || got.BackupScheduler.UUID != "scheduler-1" {
				t.Fatalf("drive=%+v err=%v", got, err)
			}
		})
	}
}

func TestDriveSchedulerActionsRejectInvalidAcknowledgements(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "redirect", status: http.StatusFound, want: "http_status"},
		{name: "unexpected success", status: http.StatusOK, body: `{}`, want: "unexpected_status"},
		{name: "malformed", status: http.StatusAccepted, body: `{"uuid":`, want: "decode_error"},
		{name: "missing uuid", status: http.StatusAccepted, body: `{"backup_scheduler":{"uuid":"scheduler-1"}}`, want: "decode_error"},
		{name: "wrong drive uuid", status: http.StatusAccepted, body: `{"uuid":"different-drive"}`, want: "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(tc.status, tc.body), nil
			})
			_, err := c.AssignDriveScheduler(context.Background(), "drive-1", "scheduler-1")
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Category != tc.want {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		name      string
		drive     string
		scheduler string
	}{
		{name: "invalid drive", drive: "bad/id", scheduler: "scheduler-1"},
		{name: "invalid scheduler", drive: "drive-1", scheduler: "bad/id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { called = true; return nil, nil })
			_, err := c.AssignDriveScheduler(context.Background(), tc.drive, tc.scheduler)
			if err == nil || called {
				t.Fatalf("err=%v called=%v", err, called)
			}
		})
	}
}

func TestDriveSchedulerActionsDecodeSchedulerURIProjection(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
		err  bool
	}{
		{name: "literal URI", body: `{"uuid":"drive-1","backup_scheduler":"/api/2.0/backupschedulers/scheduler-1/"}`, want: "scheduler-1"},
		{name: "object URI", body: `{"uuid":"drive-1","backup_scheduler":{"resource_uri":"/api/2.0/backupschedulers/scheduler-1/"}}`, want: "scheduler-1"},
		{name: "null", body: `{"uuid":"drive-1","backup_scheduler":null}`},
		{name: "wrong route", body: `{"uuid":"drive-1","backup_scheduler":"/api/2.0/drives/scheduler-1/"}`, err: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(http.StatusAccepted, tc.body), nil
			})
			got, err := c.AssignDriveScheduler(context.Background(), "drive-1", "scheduler-1")
			if tc.err {
				if err == nil {
					t.Fatal("accepted malformed scheduler URI")
				}
				return
			}
			if err != nil || got == nil {
				t.Fatalf("drive=%+v err=%v", got, err)
			}
			if tc.want == "" {
				if got.BackupScheduler != nil {
					t.Fatalf("expected absent association: %+v", got.BackupScheduler)
				}
			} else if got.BackupScheduler == nil || got.BackupScheduler.UUID != tc.want {
				t.Fatalf("drive association=%+v want=%s", got.BackupScheduler, tc.want)
			}
		})
	}
}

func TestDriveSchedulerActionTransportAndCancellation(t *testing.T) {
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("unknown outcome") })
	if _, err := c.AssignDriveScheduler(context.Background(), "drive-1", "scheduler-1"); err == nil || !strings.Contains(err.Error(), "transport") {
		t.Fatalf("transport err=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	c = remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	done := make(chan error, 1)
	go func() {
		_, err := c.AssignDriveScheduler(ctx, "drive-1", "scheduler-1")
		done <- err
	}()
	cancel()
	select {
	case err := <-done:
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Category != "cancelled" {
			t.Fatalf("cancel err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled request did not return")
	}
}
