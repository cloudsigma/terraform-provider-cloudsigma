package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestLicenses(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":"7"}],"meta":{"limit":20,"offset":0,"total_count":99,"next":"https://untrusted.invalid"}}`, true},
		{`{}`, true}, {`null`, true}, {`{"objects":null}`, true}, {`{"objects":[null,{}]}`, true}, {`{"objects":[{"name":null,"free":null}]}`, true}, {`{"objects":[],"objects":[{"name":"last"}]} {}`, true},
		{`{"objects":[{"free":1,"free":null,"initial_purchase_requirement":-0,"subsequent_purchase_requirement":1e999}]}`, true},
		{`[]`, false}, {`{"objects":{}}`, false}, {`{"objects":[{"name":1}]}`, false}, {`{"meta":{"limit":"bad"}}`, false}, {``, false},
		{`{"objects":[{"free":true,"free":"ok"}]}`, false}, {`{"objects":[{"free":{}}]}`, false}, {`{"objects":[{"initial_purchase_requirement":[]}]}`, false}, {`{"objects":[{"burstable":"true"}]}`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			calls := 0
			b := &trackedBody{Reader: strings.NewReader(tc.body)}
			c, _ := New("zrh", "fixture-user", "fixture-pass", "agent")
			c.transport = fakeDoer(func(r *http.Request) (*http.Response, error) {
				calls++
				u, p, ok := r.BasicAuth()
				if !ok || u != "fixture-user" || p != "fixture-pass" || r.URL.String() != "https://zrh.cloudsigma.com/api/2.0/licenses/" || r.URL.RawQuery != "" || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("User-Agent") != "agent" {
					t.Fatal("request")
				}
				return &http.Response{StatusCode: 200, Body: b}, nil
			})
			got, err := c.Licenses(context.Background())
			if (err == nil) != tc.valid || calls != 1 || !b.closed {
				t.Fatalf("%v calls%d", err, calls)
			}
			legacyClient := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: licenseOracleRT(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}))
			legacy, _, legacyErr := legacyClient.Licenses.List(context.Background())
			if (legacyErr == nil) != tc.valid || (tc.valid && !licenseEqual(got, legacy)) {
				t.Fatalf("pinned service oracle: %v got=%+v legacy=%+v", legacyErr, got, legacy)
			}

		})
	}
	for _, code := range []int{201, 204, 301, 404, 500} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("PRIVATE"))}, nil
		})
		if _, err := c.Licenses(context.Background()); err == nil || strings.Contains(err.Error(), "PRIVATE") {
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
		_, err := c.Licenses(ctx)
		cancel()
		if err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal(err)
		}
	}
}

// The pinned SDK model is an independent compatibility oracle, test-only.
func licenseEqual(got []License, legacy []cloudsigma.License) bool {
	if len(got) != len(legacy) {
		return false
	}
	for i := range got {
		if !reflect.DeepEqual(got[i], License(legacy[i])) {
			return false
		}
	}
	return (got == nil) == (legacy == nil)
}

type licenseOracleRT func(*http.Request) (*http.Response, error)

func (f licenseOracleRT) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func licenseReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Licenses(context.Background())
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func licenseEarlierDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	want, _ := ctx.Deadline()
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		got, _ := r.Context().Deadline()
		if !got.Equal(want) {
			t.Fatal("deadline extended")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	if _, e := c.Licenses(ctx); e != nil {
		t.Fatal(e)
	}
}

func licenseLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Licenses(ctx)
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func licenseDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Licenses(ctx)
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestLicensesBoundaries(t *testing.T) {
	t.Run("body-read", licenseReadError)
	t.Run("earlier-deadline", licenseEarlierDeadline)
	t.Run("late-cancel", licenseLateCancel)
	t.Run("deadline", licenseDeadline)
	got, err := decodeLicenses([]byte(`{"objects":[{"free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`))
	if err != nil || got[0].Free != "9007199254740993" || got[0].InitialPurchaseRequirement != "1.20e3" || got[0].SubsequentPurchaseRequirement != "-0" {
		t.Fatal(got, err)
	}
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.Licenses(context.Background()); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.Licenses(context.Background()); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
