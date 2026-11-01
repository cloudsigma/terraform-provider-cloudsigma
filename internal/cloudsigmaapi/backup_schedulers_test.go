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

func canonicalSchedulerRequest() SchedulerCreateRequest {
	return SchedulerCreateRequest{
		Type:                               "snapshot",
		Name:                               "daily snapshots",
		EnableBackupPolicyNotifications:    true,
		EnableRetentionPolicyNotifications: false,
		IsEnabled:                          true,
		IsDefault:                          false,
		ManualIncrementalBackup:            "00:00",
		RetentionPolicy: SchedulerRetentionPolicy{
			Name:  "keep 7 days",
			Rules: []SchedulerRetentionRule{{Period: "days", Quantity: 7}},
		},
		IncrementalBackup: SchedulerIncrementalBackup{
			DayOfMonth: "*",
			DayOfWeek:  []string{"*"},
			Month:      "*",
			Hour:       "1",
			Minute:     "0",
			Repeat:     &SchedulerRepeat{},
			StartTime:  &SchedulerClock{Hour: 0, Minute: 0},
			EndTime:    &SchedulerClock{Hour: 23, Minute: 45},
		},
		UserTimezone:         "Etc/UTC",
		OrchestratorTimezone: "Etc/UTC",
		Tags:                 []SchedulerTagRef{{UUID: "tag-1"}},
		Meta:                 map[string]any{"owner": "fixture"},
	}
}

func TestCreateBackupSchedulerProjectsCanonicalWritableFields(t *testing.T) {
	var request map[string]any
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/2.0/backupschedulers/" {
			t.Fatalf("route=%s %s", req.Method, req.URL.Path)
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
			t.Fatalf("auth=%q", req.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(req.Body)
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatal(err)
		}
		return remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"scheduler-1","resource_uri":"/api/2.0/backupschedulers/scheduler-1/","name":"daily snapshots","type":"snapshot","owner":{"uuid":"response-only"},"permissions":[],"grantees":[]}]}`), nil
	})
	got, err := c.CreateBackupScheduler(context.Background(), canonicalSchedulerRequest())
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "scheduler-1" || got.Name != "daily snapshots" || got.Type != "snapshot" {
		t.Fatalf("scheduler=%+v", got)
	}
	objects, ok := request["objects"].([]any)
	if !ok || len(objects) != 1 {
		t.Fatalf("request=%#v", request)
	}
	object := objects[0].(map[string]any)
	for _, forbidden := range []string{"uuid", "resource_uri", "owner", "permissions", "grantees"} {
		if _, ok := object[forbidden]; ok {
			t.Fatalf("response-only field %q projected: %#v", forbidden, object)
		}
	}
	if object["type"] != "snapshot" || object["name"] != "daily snapshots" || object["user_timezone"] != "Etc/UTC" || object["orchestrator_timezone"] != "Etc/UTC" {
		t.Fatalf("projection=%#v", object)
	}
	tags, ok := object["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0].(map[string]any)["uuid"] != "tag-1" {
		t.Fatalf("tags=%#v", object["tags"])
	}
	retention := object["retention_policy"].(map[string]any)
	if retention["rules"].([]any)[0].(map[string]any)["period"] != "days" {
		t.Fatalf("retention=%#v", retention)
	}
}

func TestCreateBackupSchedulerSupportsBearerAndMonthList(t *testing.T) {
	input := canonicalSchedulerRequest()
	input.IncrementalBackup.Month = []string{"1", "3", "12"}
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("auth=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"scheduler-token"}]}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if got, err := c.CreateBackupScheduler(context.Background(), input); err != nil || got.UUID != "scheduler-token" {
		t.Fatalf("scheduler=%+v err=%v", got, err)
	}
}

func TestCreateBackupSchedulerValidationAndFailures(t *testing.T) {
	for _, bad := range []SchedulerCreateRequest{
		{},
		func() SchedulerCreateRequest { v := canonicalSchedulerRequest(); v.Type = "unsupported"; return v }(),
		func() SchedulerCreateRequest {
			v := canonicalSchedulerRequest()
			v.ManualIncrementalBackup = ""
			return v
		}(),
		func() SchedulerCreateRequest {
			v := canonicalSchedulerRequest()
			v.RetentionPolicy.Rules[0].Period = "years"
			return v
		}(),
		func() SchedulerCreateRequest {
			v := canonicalSchedulerRequest()
			v.RetentionPolicy.Rules[0].Quantity = 0
			return v
		}(),
		func() SchedulerCreateRequest {
			v := canonicalSchedulerRequest()
			v.IncrementalBackup.Month = map[string]any{"bad": true}
			return v
		}(),
		func() SchedulerCreateRequest {
			v := canonicalSchedulerRequest()
			v.Tags = []SchedulerTagRef{{UUID: "bad/id"}}
			return v
		}(),
		func() SchedulerCreateRequest {
			v := canonicalSchedulerRequest()
			v.Meta = map[string]any{"owner": 42}
			return v
		}(),
		func() SchedulerCreateRequest {
			v := canonicalSchedulerRequest()
			v.Meta = map[string]any{"this-key-is-longer-than-thirty-two-characters": "x"}
			return v
		}(),
	} {
		c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid request reached transport")
			return nil, nil
		})
		if _, err := c.CreateBackupScheduler(context.Background(), bad); err == nil {
			t.Fatal("invalid scheduler accepted")
		}
	}
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"redirect", http.StatusFound, "", "http_status"},
		{"unexpected", http.StatusAccepted, "", "unexpected_status"},
		{"malformed", http.StatusCreated, `{"objects":[`, "decode_error"},
		{"missing uuid", http.StatusCreated, `{"objects":[{"name":"x"}]}`, "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return remoteSnapshotResponse(tc.status, tc.body), nil })
			_, err := c.CreateBackupScheduler(context.Background(), canonicalSchedulerRequest())
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Category != tc.want {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
		t.Fatal("cancelled request reached transport")
		return nil, nil
	})
	_, err := c.CreateBackupScheduler(ctx, canonicalSchedulerRequest())
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("cancel err=%v", err)
	}
}

func TestCreateBackupSchedulerPreservesServerManualBackupGrammar(t *testing.T) {
	input := canonicalSchedulerRequest()
	input.ManualIncrementalBackup = "*,00:00,12:30"
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		var document struct {
			Objects []map[string]any `json:"objects"`
		}
		if err := json.NewDecoder(req.Body).Decode(&document); err != nil {
			t.Fatal(err)
		}
		if got := document.Objects[0]["manual_incremental_backup"]; got != input.ManualIncrementalBackup {
			t.Fatalf("manual_incremental_backup=%#v", got)
		}
		return remoteSnapshotResponse(http.StatusCreated, `{"objects":[{"uuid":"scheduler-manual"}]}`), nil
	})
	if got, err := c.CreateBackupScheduler(context.Background(), input); err != nil || got.UUID != "scheduler-manual" {
		t.Fatalf("scheduler=%+v err=%v", got, err)
	}
}

func TestUpdateBackupSchedulerProjectsWritableFieldsAndAuth(t *testing.T) {
	input := canonicalSchedulerRequest()
	input.InitialBackup = &SchedulerInitialBackup{
		DayOfWeek: []string{"*"},
		StartTime: &SchedulerClock{Hour: 0, Minute: 0},
		EndTime:   &SchedulerClock{Hour: 23, Minute: 45},
	}
	var request map[string]any
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || req.URL.Path != "/api/2.0/backupschedulers/scheduler-1/" {
			t.Fatalf("route=%s %s", req.Method, req.URL.Path)
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
			t.Fatalf("auth=%q", req.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		return remoteSnapshotResponse(http.StatusOK, `{"uuid":"scheduler-1","name":"updated","type":"snapshot","owner":{"uuid":"response-only"},"permissions":[],"grantees":[]}`), nil
	})
	got, err := c.UpdateBackupScheduler(context.Background(), "scheduler-1", input)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "scheduler-1" || got.Name != "updated" || got.Type != "snapshot" {
		t.Fatalf("scheduler=%+v", got)
	}
	for _, forbidden := range []string{"uuid", "resource_uri", "owner", "permissions", "grantees"} {
		if _, ok := request[forbidden]; ok {
			t.Fatalf("response-only field %q projected: %#v", forbidden, request)
		}
	}
	if request["name"] != input.Name || request["manual_incremental_backup"] != input.ManualIncrementalBackup {
		t.Fatalf("projection=%#v", request)
	}
	if _, ok := request["initial_backup"].(map[string]any); !ok {
		t.Fatalf("initial_backup missing from full PUT projection: %#v", request)
	}
	if _, ok := request["incremental_backup"].(map[string]any); !ok {
		t.Fatalf("incremental_backup missing from full PUT projection: %#v", request)
	}

	c = remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		return remoteSnapshotResponse(http.StatusOK, `{"uuid":"scheduler-token","name":"updated"}`), nil
	})
	c.username, c.password, c.token = "", "", "fixture-token"
	if got, err := c.UpdateBackupScheduler(context.Background(), "scheduler-token", input); err != nil || got.UUID != "scheduler-token" {
		t.Fatalf("scheduler=%+v err=%v", got, err)
	}
}

func TestUpdateBackupSchedulerRejectsInvalidAcknowledgements(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "redirect", status: http.StatusFound, want: "http_status"},
		{name: "unexpected success", status: http.StatusAccepted, body: `{"uuid":"scheduler-1"}`, want: "unexpected_status"},
		{name: "malformed", status: http.StatusOK, body: `{"uuid":`, want: "decode_error"},
		{name: "wrong uuid", status: http.StatusOK, body: `{"uuid":"different"}`, want: "decode_error"},
		{name: "trailing data", status: http.StatusOK, body: `{"uuid":"scheduler-1"}{}`, want: "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(tc.status, tc.body), nil
			})
			_, err := c.UpdateBackupScheduler(context.Background(), "scheduler-1", canonicalSchedulerRequest())
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Category != tc.want {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	for _, uuid := range []string{"bad/id", ""} {
		c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid uuid reached transport")
			return nil, nil
		})
		if _, err := c.UpdateBackupScheduler(context.Background(), uuid, canonicalSchedulerRequest()); err == nil {
			t.Fatalf("invalid uuid %q accepted", uuid)
		}
	}
}

func TestUpdateBackupSchedulerContextAndTransportErrors(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() context.Context
		want string
	}{
		{name: "cancelled before request", ctx: func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}, want: "cancelled"},
		{name: "deadline before request", ctx: func() context.Context {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			return ctx
		}, want: "timeout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
				return nil, req.Context().Err()
			})
			_, err := c.UpdateBackupScheduler(tc.ctx(), "scheduler-1", canonicalSchedulerRequest())
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Category != tc.want {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("unknown outcome")
	})
	_, err := c.UpdateBackupScheduler(context.Background(), "scheduler-1", canonicalSchedulerRequest())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Category != "transport" {
		t.Fatalf("transport err=%v", err)
	}
}

func TestDeleteBackupSchedulerRequiresNoContentAndUsesAuth(t *testing.T) {
	for _, bearer := range []bool{false, true} {
		t.Run(func() string {
			if bearer {
				return "bearer"
			}
			return "basic"
		}(), func(t *testing.T) {
			c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodDelete || req.URL.Path != "/api/2.0/backupschedulers/scheduler-1/" {
					t.Fatalf("route=%s %s", req.Method, req.URL.Path)
				}
				if bearer {
					if req.Header.Get("Authorization") != "Bearer fixture-token" {
						t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
					}
				} else if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
					t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
				}
				return remoteSnapshotResponse(http.StatusNoContent, ""), nil
			})
			if bearer {
				c.username, c.password, c.token = "", "", "fixture-token"
			}
			if err := c.DeleteBackupScheduler(context.Background(), "scheduler-1"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeleteBackupSchedulerRejectsFailuresAndContextErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		want   string
	}{
		{name: "redirect", status: http.StatusFound, want: "http_status"},
		{name: "unexpected success", status: http.StatusOK, want: "unexpected_status"},
		{name: "server error", status: http.StatusInternalServerError, want: "http_status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(tc.status, ""), nil
			})
			var apiErr *Error
			if err := c.DeleteBackupScheduler(context.Background(), "scheduler-1"); !errors.As(err, &apiErr) || apiErr.Category != tc.want {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	for _, uuid := range []string{"bad/id", ""} {
		c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid uuid reached transport")
			return nil, nil
		})
		if err := c.DeleteBackupScheduler(context.Background(), uuid); err == nil {
			t.Fatalf("invalid uuid %q accepted", uuid)
		}
	}
	for _, tc := range []struct {
		name string
		ctx  func() context.Context
		want string
	}{
		{name: "cancelled", ctx: func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}, want: "cancelled"},
		{name: "deadline", ctx: func() context.Context {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			return ctx
		}, want: "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) { return nil, req.Context().Err() })
			var apiErr *Error
			if err := c.DeleteBackupScheduler(tc.ctx(), "scheduler-1"); !errors.As(err, &apiErr) || apiErr.Category != tc.want {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("unknown outcome") })
	var apiErr *Error
	if err := c.DeleteBackupScheduler(context.Background(), "scheduler-1"); !errors.As(err, &apiErr) || apiErr.Category != "transport" {
		t.Fatalf("transport err=%v", err)
	}
}
