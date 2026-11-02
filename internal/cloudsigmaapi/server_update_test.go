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

func intPtr(v int) *int                                       { return &v }
func stringPtr(v string) *string                              { return &v }
func mapPtr(v map[string]interface{}) *map[string]interface{} { return &v }

func TestUpdateServerProjectsWritableCoreAndChecksIdentity(t *testing.T) {
	var request map[string]any
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/2.0/servers/server-1/" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"server-1","name":"updated","cpu":2000,"mem":536870912,"status":"stopped","owner":{"uuid":"owner"},"jobs":[{"uuid":"job"}],"drives":[{"drive":{"uuid":"drive"}}],"nics":[{"mac":"aa:bb"}]}`))}, nil
	})
	got, err := c.UpdateServer(context.Background(), "server-1", ServerUpdateRequest{CPU: intPtr(2000), Memory: intPtr(536870912), Name: stringPtr("updated"), SMP: intPtr(2), VNCPassword: stringPtr("secret"), Meta: mapPtr(map[string]interface{}{"managed": true})})
	if err != nil || got.UUID != "server-1" || got.Status != "stopped" || len(got.Drives) != 1 {
		t.Fatalf("server=%+v err=%v", got, err)
	}
	want := map[string]any{"cpu": float64(2000), "mem": float64(536870912), "meta": map[string]any{"managed": true}, "name": "updated", "smp": float64(2), "vnc_password": "secret"}
	if !reflect.DeepEqual(request, want) {
		t.Fatalf("request projection=%#v want=%#v", request, want)
	}
}

func TestUpdateServerRejectsUnexpectedResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "redirect", status: http.StatusTemporaryRedirect, body: "PRIVATE", want: "http_status"},
		{name: "accepted", status: http.StatusAccepted, body: `{"uuid":"server-1"}`, want: "unexpected_status"},
		{name: "malformed", status: http.StatusOK, body: `{`, want: "decode_error"},
		{name: "identity", status: http.StatusOK, body: `{"uuid":"other"}`, want: "decode_error"},
		{name: "trailing", status: http.StatusOK, body: `{"uuid":"server-1"} {}`, want: "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			_, err := c.UpdateServer(context.Background(), "server-1", ServerUpdateRequest{Name: stringPtr("updated")})
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
}

func TestUpdateServerContextAndTransportFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("PRIVATE") })
	if _, err := c.UpdateServer(ctx, "server-1", ServerUpdateRequest{Name: stringPtr("updated")}); err == nil || err.Error() != "cancelled" || calls != 0 {
		t.Fatalf("pre-cancel err=%v calls=%d", err, calls)
	}
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) { return nil, errors.New("PRIVATE transport") })
	if _, err := c.UpdateServer(context.Background(), "server-1", ServerUpdateRequest{Name: stringPtr("updated")}); err == nil || err.Error() != "transport" || strings.Contains(err.Error(), "PRIVATE") {
		t.Fatalf("transport err=%v", err)
	}
	deadline, end := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer end()
	if _, err := c.UpdateServer(deadline, "server-1", ServerUpdateRequest{Name: stringPtr("updated")}); err == nil || err.Error() != "timeout" {
		t.Fatalf("deadline err=%v", err)
	}
	if _, err := c.UpdateServer(context.Background(), "../bad", ServerUpdateRequest{Name: stringPtr("updated")}); err == nil {
		t.Fatal("accepted invalid uuid")
	}
}

func TestUpdateServerCanExplicitlyClearMetadataAndSetZeroValues(t *testing.T) {
	var request map[string]any
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"server-1"}`))}, nil
	})
	empty := map[string]interface{}{}
	zero := 0
	blank := ""
	_, err := c.UpdateServer(context.Background(), "server-1", ServerUpdateRequest{CPU: &zero, Memory: &zero, Name: &blank, SMP: &zero, VNCPassword: &blank, Meta: &empty})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"cpu": float64(0), "mem": float64(0), "name": "", "smp": float64(0), "vnc_password": "", "meta": map[string]any{}}
	if !reflect.DeepEqual(request, want) {
		t.Fatalf("request=%#v want=%#v", request, want)
	}
}
