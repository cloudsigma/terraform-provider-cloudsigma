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

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestLocations(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"objects":[{"id":"ZRH","api_endpoint":"a","country_code":"CH","display_name":"Zurich","alternative_frontend_url":"b","default_frontend_signup_url":"c","default_frontend_url":"d","documentation_url":"e","upload_url":"f","websocket_url":"g"}],"meta":{"limit":20,"offset":0,"total_count":99,"next":"https://untrusted.invalid"}}`, true},
		{`{}`, true}, {`null`, true}, {`{"objects":null}`, true}, {`{"objects":[null,{}]}`, true}, {`{"objects":[{"id":null}]}`, true}, {`{"objects":[],"objects":[{"id":"last"}]} {}`, true}, {`[]`, false}, {`{"objects":{}}`, false}, {`{"objects":[{"id":1}]}`, false}, {`{"meta":{"limit":"bad"}}`, false}, {``, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			calls := 0
			b := &trackedBody{Reader: strings.NewReader(tc.body)}
			c, _ := New("zrh", "fixture-user", "fixture-pass", "agent")
			c.transport = fakeDoer(func(r *http.Request) (*http.Response, error) {
				calls++
				u, p, ok := r.BasicAuth()
				if !ok || u != "fixture-user" || p != "fixture-pass" || r.URL.String() != "https://zrh.cloudsigma.com/api/2.0/locations/" || r.URL.RawQuery != "" || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("User-Agent") != "agent" {
					t.Fatal("request")
				}
				return &http.Response{StatusCode: 200, Body: b}, nil
			})
			got, err := c.Locations(context.Background())
			if (err == nil) != tc.valid || calls != 1 || !b.closed {
				t.Fatalf("%v calls%d", err, calls)
			}
			if tc.valid {
				var legacy struct {
					Objects []cloudsigma.Location `json:"objects"`
				}
				if e := json.NewDecoder(strings.NewReader(tc.body)).Decode(&legacy); e != nil || !locationEqual(got, legacy.Objects) {
					t.Fatal("compatibility")
				}
			}
		})
	}
	for _, code := range []int{201, 204, 301, 404, 500} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("PRIVATE"))}, nil
		})
		if _, err := c.Locations(context.Background()); err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal(err)
		}
	}
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if cancelled {
			cancel()
		}
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			if cancelled {
				return nil, r.Context().Err()
			}
			return nil, errors.New("PRIVATE")
		})
		_, err := c.Locations(ctx)
		cancel()
		if err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal(err)
		}
	}
}

// The pinned SDK model is an independent compatibility oracle, test-only.
func locationEqual(got []Location, legacy []cloudsigma.Location) bool {
	if len(got) != len(legacy) {
		return false
	}
	for i := range got {
		if !reflect.DeepEqual(got[i], Location(legacy[i])) {
			return false
		}
	}
	return (got == nil) == (legacy == nil)
}
