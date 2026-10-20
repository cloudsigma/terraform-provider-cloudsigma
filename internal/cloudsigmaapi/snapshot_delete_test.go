package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestSnapshotDelete(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 205, 301, 302, 307, 308, 400, 401, 403, 404, 409, 429, 500} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			calls := 0
			bodies := []*trackedBody{}
			rt := func(q *http.Request) (*http.Response, error) {
				calls++
				if q.Body != nil {
					payload, _ := io.ReadAll(q.Body)
					if len(payload) != 0 {
						t.Fatal("unexpected body")
					}
				}
				if q.Method != "DELETE" || q.URL.Path != "/api/2.0/snapshots/fixture/" || q.URL.RawQuery != "" {
					t.Fatal(q)
				}
				b := &trackedBody{Reader: strings.NewReader("PRIVATE")}
				bodies = append(bodies, b)
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: b, Request: q}, nil
			}
			c := reviewNativeNew(rt)
			err := c.DeleteSnapshot(context.Background(), "fixture")
			if (err == nil) != (code == 204) {
				t.Fatal(code, err)
			}
			if err != nil {
				var e *Error
				if !errors.As(err, &e) || e.Status != code || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal(err)
				}
			}
			old := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
			_, oe := old.Snapshots.Delete(context.Background(), "fixture")
			if (oe == nil) != (code >= 200 && code < 300) || calls != 2 {
				t.Fatal(oe, calls)
			}
			for _, b := range bodies {
				if !b.closed {
					t.Fatal("unclosed")
				}
			}
		})
	}
}
func TestSnapshotDeleteBoundaries(t *testing.T) {
	for _, id := range []string{"", "../x", "a/b", "a?b", "a#b", "a%2fb", "a\n"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("dispatch"); return nil, nil })
		if c.DeleteSnapshot(context.Background(), id) == nil {
			t.Fatal(id)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("pre-cancel dispatch"); return nil, nil })
	if e := c.DeleteSnapshot(ctx, "fixture"); e == nil || e.Error() != "cancelled" {
		t.Fatal(e)
	}
	for _, late := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		b := &trackedBody{Reader: strings.NewReader("")}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			calls++
			cancel()
			if late {
				return &http.Response{StatusCode: 204, Body: b}, nil
			}
			return nil, context.Canceled
		})
		if e := c.DeleteSnapshot(ctx, "fixture"); e == nil || e.Error() != "cancelled" || calls != 1 {
			t.Fatal(e, calls)
		}
		if late && !b.closed {
			t.Fatal("unclosed")
		}
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if e := c.DeleteSnapshot(ctx, "fixture"); e == nil || e.Error() != "timeout" {
		t.Fatal(e)
	}
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })
	if e := c.DeleteSnapshot(context.Background(), "fixture"); e == nil || e.Error() != "transport" {
		t.Fatal(e)
	}
}

// Characterize the held callbacks' exact SDK requests, not native coverage.
func TestSnapshotHeldMutationOracle(t *testing.T) {
	for _, create := range []bool{true, false} {
		calls := 0
		old := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(func(q *http.Request) (*http.Response, error) {
			calls++
			b, _ := io.ReadAll(q.Body)
			s := string(b)
			if !strings.Contains(s, `"drive":{"`) || !strings.Contains(s, `"uuid":"drive"`) || !strings.Contains(s, `"tags":[]`) || strings.Contains(s, `"meta"`) || strings.Contains(s, `"name"`) {
				t.Fatal(s)
			}
			if create {
				if q.Method != "POST" || q.URL.Path != "/api/2.0/snapshots/" || !strings.Contains(s, `"objects":[`) {
					t.Fatal(q, s)
				}
			} else {
				if q.Method != "PUT" || q.URL.Path != "/api/2.0/snapshots/fixture/" || strings.Contains(s, `"objects"`) {
					t.Fatal(q, s)
				}
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"objects":[],"drive":{"uuid":"drive"}}`)), Request: q}, nil
		})}))
		snap := cloudsigma.Snapshot{Drive: &cloudsigma.Drive{UUID: "drive"}, Name: ""}
		if create {
			_, _, e := old.Snapshots.Create(context.Background(), &cloudsigma.SnapshotCreateRequest{Snapshots: []cloudsigma.Snapshot{snap}})
			if e != nil {
				t.Fatal(e)
			}
		} else {
			_, _, e := old.Snapshots.Update(context.Background(), "fixture", &cloudsigma.SnapshotUpdateRequest{Snapshot: &snap})
			if e != nil {
				t.Fatal(e)
			}
		}
		if calls != 1 {
			t.Fatal(calls)
		}
	}
}
