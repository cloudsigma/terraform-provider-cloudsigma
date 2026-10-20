package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestDriveCreate(t *testing.T) {
	for _, storage := range []string{"", "nvme"} {
		for _, code := range []int{200, 201, 202, 204, 301, 401, 404, 409, 429, 500} {
			t.Run(fmt.Sprintf("%s/%d", storage, code), func(t *testing.T) {
				var bodies []map[string]interface{}
				rt := func(r *http.Request) (*http.Response, error) {
					if r.Method != "POST" || r.URL.RequestURI() != "/api/2.0/drives/" {
						t.Fatal("route", r.URL)
					}
					if _, ok := r.Context().Deadline(); ok {
						t.Fatal("new deadline")
					}
					var body map[string]interface{}
					if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
						t.Fatal(e)
					}
					bodies = append(bodies, body)
					return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"created","status":"creating"}]}`))}, nil
				}
				c := reviewNativeNew(rt)
				h := &http.Client{Transport: subscriptionOracleRT(rt), Timeout: time.Nanosecond}
				c.transport = h
				got, e := c.CreateDrive(context.Background(), DriveCreateRequest{Media: "disk", Name: "fixture", Size: 1073741824, StorageType: storage})
				sdk := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt)}))
				old, _, oe := sdk.Drives.Create(context.Background(), &cs.DriveCreateRequest{Drives: []cs.Drive{{Media: "disk", Name: "fixture", Size: 1073741824, StorageType: storage}}})
				if len(bodies) != 2 || !reflect.DeepEqual(bodies[0], bodies[1]) || h.Timeout != time.Nanosecond {
					t.Fatal("wire/timeout", bodies)
				}
				fields := map[string]interface{}{"media": "disk", "name": "fixture", "size": float64(1073741824)}
				if storage != "" {
					fields["storage_type"] = storage
				}
				if !reflect.DeepEqual(bodies[0], map[string]interface{}{"objects": []interface{}{fields}}) {
					t.Fatal("independent writable fields", bodies)
				}
				if (e == nil) != (code == 201) || (oe == nil) != (code >= 200 && code < 300) {
					t.Fatal(e, oe)
				}
				if e == nil && (got.UUID != "created" || got.UUID != old[0].UUID) {
					t.Fatal(got)
				}
			})
		}
	}
	for _, body := range []string{`{}`, `{"objects":[]}`, `{"objects":null}`, `invalid`, `{"objects":{}}`} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 201, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, e := c.CreateDrive(context.Background(), DriveCreateRequest{}); e == nil {
			t.Fatal("accepted invalid response", body)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { calls++; return nil, fmt.Errorf("PRIVATE") })
	if _, e := c.CreateDrive(ctx, DriveCreateRequest{}); e == nil || calls != 0 {
		t.Fatal(e, calls)
	}
}
