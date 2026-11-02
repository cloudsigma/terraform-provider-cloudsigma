package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestUpdateDriveProjectsWritableFieldsAndChecksIdentity(t *testing.T) {
	var body map[string]any
	c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || req.URL.Path != "/api/2.0/drives/drive-1/" {
			t.Fatalf("request=%s %s", req.Method, req.URL)
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"drive-1","name":"updated","status":"unmounted","owner":{"uuid":"owner"},"runtime":{"storage_type":"nvme"},"tags":[{"uuid":"tag"}]}`)), Request: req}, nil
	})
	got, err := c.UpdateDrive(context.Background(), "drive-1", DriveUpdateRequest{
		AllowMultimount: boolPtr(true),
		Media:           "disk",
		Meta:            map[string]interface{}{"managed": true},
		Name:            "updated",
		Size:            1073741824,
		StorageType:     "nvme",
		Tags:            []Tag{{UUID: "tag"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if body["uuid"] != nil || body["status"] != nil || body["owner"] != nil || body["runtime"] != nil {
		t.Fatalf("response-only fields projected: %#v", body)
	}
	if got == nil || got.UUID != "drive-1" || got.Name != "updated" || got.Status != "unmounted" || got.Owner == nil || got.Owner.UUID != "owner" {
		t.Fatalf("response=%+v", got)
	}
}

func TestUpdateDrivePreservesExplicitFalseAndEmptyClears(t *testing.T) {
	var body map[string]any
	c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"drive-1"}`)), Request: req}, nil
	})
	emptyMeta := map[string]interface{}{}
	emptyTags := []Tag{}
	if _, err := c.UpdateDrive(context.Background(), "drive-1", DriveUpdateRequest{
		AllowMultimount: boolPtr(false), Meta: emptyMeta, Tags: emptyTags,
	}); err != nil {
		t.Fatal(err)
	}
	if value, ok := body["allow_multimount"]; !ok || value != false {
		t.Fatalf("explicit false omitted: %#v", body)
	}
	if value, ok := body["meta"]; !ok || value == nil {
		t.Fatalf("explicit metadata clear omitted: %#v", body)
	}
	if value, ok := body["tags"]; !ok || value == nil {
		t.Fatalf("explicit tag clear omitted: %#v", body)
	}
}

func TestUpdateDriveRejectsUnexpectedStatusesAndMalformedResponses(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{name: "created", code: http.StatusCreated, body: `{"uuid":"drive-1"}`},
		{name: "redirect", code: http.StatusFound, body: ``},
		{name: "malformed", code: http.StatusOK, body: `{}`},
		{name: "wrong identity", code: http.StatusOK, body: `{"uuid":"other"}`},
		{name: "trailing", code: http.StatusOK, body: `{"uuid":"drive-1"} {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			if got, err := c.UpdateDrive(context.Background(), "drive-1", DriveUpdateRequest{Name: "name"}); got != nil || err == nil {
				t.Fatalf("got=%v err=%v", got, err)
			}
		})
	}
}

func boolPtr(value bool) *bool { return &value }

func TestUpdateDriveRejectsInvalidUUIDAndHonorsCancellation(t *testing.T) {
	calls := 0
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, fmt.Errorf("unexpected request")
	})
	if _, err := c.UpdateDrive(context.Background(), "../drive", DriveUpdateRequest{}); err == nil || calls != 0 {
		t.Fatalf("invalid uuid err=%v calls=%d", err, calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.UpdateDrive(ctx, "drive-1", DriveUpdateRequest{}); err == nil || calls != 0 {
		t.Fatalf("cancel err=%v calls=%d", err, calls)
	}
}
