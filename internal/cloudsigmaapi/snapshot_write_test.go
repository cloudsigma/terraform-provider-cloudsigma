package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCreateSnapshotProjectsWritableFields(t *testing.T) {
	var request map[string]any
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/2.0/snapshots/" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"snapshot-1","drive":{"uuid":"drive-1"},"name":"taken","status":"creating","drive_meta":{"private":"ignored"}}]}`))}, nil
	})
	got, err := c.CreateSnapshot(context.Background(), SnapshotCreateRequest{Drive: SnapshotDriveRef{UUID: "drive-1"}, Name: "requested"})
	if err != nil || got.UUID != "snapshot-1" {
		t.Fatalf("snapshot=%+v err=%v", got, err)
	}
	want := map[string]any{"objects": []any{map[string]any{"drive": map[string]any{"uuid": "drive-1"}, "name": "requested"}}}
	if !reflect.DeepEqual(request, want) {
		t.Fatalf("request projection: %#v", request)
	}
}

func TestUpdateSnapshotProjectsWritableFieldsAndChecksIdentity(t *testing.T) {
	var request map[string]any
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/2.0/snapshots/snapshot-1/" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"snapshot-1","drive":{"uuid":"drive-1"},"name":"updated","status":"available"}`))}, nil
	})
	got, err := c.UpdateSnapshot(context.Background(), "snapshot-1", SnapshotUpdateRequest{Drive: SnapshotDriveRef{UUID: "drive-1"}, Name: "updated"})
	if err != nil || got.UUID != "snapshot-1" {
		t.Fatalf("snapshot=%+v err=%v", got, err)
	}
	want := map[string]any{"drive": map[string]any{"uuid": "drive-1"}, "name": "updated"}
	if !reflect.DeepEqual(request, want) {
		t.Fatalf("request projection: %#v", request)
	}

	c = reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"other"}`))}, nil
	})
	if _, err := c.UpdateSnapshot(context.Background(), "snapshot-1", SnapshotUpdateRequest{Drive: SnapshotDriveRef{UUID: "drive-1"}}); err == nil || err.Error() != "decode_error status=200" {
		t.Fatalf("identity mismatch: %v", err)
	}
}

func TestSnapshotWriteRejectsUnexpectedAndMalformedResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "redirect", status: http.StatusTemporaryRedirect, body: "PRIVATE", want: "http_status"},
		{name: "ok status", status: http.StatusOK, body: `{"objects":[{"uuid":"snapshot-1"}]}`, want: "unexpected_status"},
		{name: "accepted", status: http.StatusAccepted, body: "PRIVATE", want: "unexpected_status"},
		{name: "malformed", status: http.StatusCreated, body: `{`, want: "decode_error"},
		{name: "wrong count", status: http.StatusCreated, body: `{"objects":[]}`, want: "decode_error"},
		{name: "missing identity", status: http.StatusCreated, body: `{"objects":[{}]}`, want: "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			_, err := c.CreateSnapshot(context.Background(), SnapshotCreateRequest{Drive: SnapshotDriveRef{UUID: "drive"}})
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	for _, input := range []SnapshotCreateRequest{{}, {Drive: SnapshotDriveRef{UUID: "../bad"}}} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("dispatch"); return nil, nil })
		if _, err := c.CreateSnapshot(context.Background(), input); err == nil {
			t.Fatalf("accepted invalid input: %+v", input)
		}
	}
}

func TestSnapshotWriteContextAndTransportFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("PRIVATE") })
	if _, err := c.CreateSnapshot(ctx, SnapshotCreateRequest{Drive: SnapshotDriveRef{UUID: "drive"}}); err == nil || err.Error() != "cancelled" || calls != 0 {
		t.Fatalf("pre-cancel err=%v calls=%d", err, calls)
	}
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) { return nil, errors.New("PRIVATE transport") })
	if _, err := c.CreateSnapshot(context.Background(), SnapshotCreateRequest{Drive: SnapshotDriveRef{UUID: "drive"}}); err == nil || err.Error() != "transport" || strings.Contains(err.Error(), "PRIVATE") {
		t.Fatalf("transport err=%v", err)
	}
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"ok"}]}`))}, nil
	})
	deadline, end := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer end()
	if _, err := c.CreateSnapshot(deadline, SnapshotCreateRequest{Drive: SnapshotDriveRef{UUID: "drive"}}); err == nil || err.Error() != "timeout" {
		t.Fatalf("deadline err=%v", err)
	}
}

func TestSnapshotWriteImposesBoundedRequestDeadline(t *testing.T) {
	c := reviewNativeNew(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		if !ok || time.Until(deadline) > requestTimeout {
			t.Fatalf("deadline missing or too long: %v", deadline)
		}
		return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"bounded"}]}`))}, nil
	})
	if _, err := c.CreateSnapshot(context.Background(), SnapshotCreateRequest{Drive: SnapshotDriveRef{UUID: "drive"}}); err != nil {
		t.Fatal(err)
	}
}
